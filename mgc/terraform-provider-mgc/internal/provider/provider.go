package provider

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"magalu.cloud/core/auth"
	mgcSdk "magalu.cloud/sdk"
)

var _ provider.Provider = (*MgcProvider)(nil)

const providerTypeName = "mgc"

var ignoredTFModules = []string{"profile"}

type apiSpec struct {
	name    string
	version string
}

func (a apiSpec) String() string {
	return fmt.Sprintf("%s@%s", a.name, a.version)
}

func parseApiSpec(spec string) *apiSpec {
	parts := strings.Split(spec, "@")
	name := parts[0]
	version := parts[1]
	return &apiSpec{name, version}
}

type MgcProvider struct {
	version string
	commit  string
	date    string
	sdk     *mgcSdk.Sdk
	apis    []*apiSpec
}

type MgcProviderModel struct {
	RefreshToken types.String   `tfsdk:"refresh_token"`
	AccessToken  types.String   `tfsdk:"access_token"`
	Apis         []types.String `tfsdk:"apis"`
}

func (p *MgcProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	tflog.Debug(ctx, "setting provider metadata")
	resp.TypeName = providerTypeName
	resp.Version = p.version
}

func (p *MgcProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	tflog.Debug(ctx, "setting provider schema")
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"refresh_token": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Refresh Token used to authenticate with the MagaluID platform.",
			},
			"access_token": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Access Token used to authenticate with the MagaluID platform.",
			},
			"apis": schema.ListAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: "Which MagaluCloud products need to be supported by the provider.",
				// Should we use validators to know if the apis exist? Or return an error later
				// Validators: ,
			},
		},
	}
}

func (p *MgcProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Info(ctx, "configuring MGC provider")
	var config MgcProviderModel

	// Load all configurations
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)

	refreshToken := config.RefreshToken.ValueString()
	accessToken := config.AccessToken.ValueString()
	if accessToken != "" || refreshToken != "" {
		_ = p.sdk.Auth().SetTokens(&auth.LoginResult{AccessToken: accessToken, RefreshToken: refreshToken})
	}

	if len(config.Apis) == 0 {
		p.attrErr(resp, "apis")
		return
	}
	p.apis = make([]*apiSpec, len(config.Apis))
	for i, spec := range config.Apis {
		p.apis[i] = parseApiSpec(spec.ValueString())
	}
	tflog.Debug(ctx, "using `apis` property from tf file")

	resp.DataSourceData = p.sdk
	resp.ResourceData = p.sdk
}

func (p *MgcProvider) Resources(ctx context.Context) []func() resource.Resource {
	tflog.Info(ctx, "configuring MGC provider resources")

	root := p.sdk.Group()
	resources, err := collectGroupResources(ctx, p.sdk, root, []string{providerTypeName})
	if err != nil {
		tflog.Error(ctx, fmt.Sprintf("An error occurred while generating the provider resource list: %v", err))
	}

	return resources
}

func collectGroupResources(
	ctx context.Context,
	sdk *mgcSdk.Sdk,
	group mgcSdk.Grouper,
	path []string,
) ([]func() resource.Resource, error) {
	// TODO: We should check if the version is correct in the Configuration call or Resource
	debugMap := map[string]any{"path": path}
	tflog.Debug(ctx, "Collecting resources", debugMap)
	var resources []func() resource.Resource
	var create, read, update, delete, list mgcSdk.Executor
	var connectionExecs []mgcSdk.Executor
	_, err := group.VisitChildren(func(child mgcSdk.Descriptor) (run bool, err error) {
		if group.IsInternal() {
			return true, nil
		}

		if childGroup, ok := child.(mgcSdk.Grouper); ok {
			if slices.Contains(ignoredTFModules, childGroup.Name()) {
				return true, nil
			}

			oldLen := len(path)
			path = append(path, childGroup.Name())
			childResources, err := collectGroupResources(ctx, sdk, childGroup, path)
			resources = append(resources, childResources...)
			path = path[:oldLen]
			return true, err
		} else if exec, ok := child.(mgcSdk.Executor); ok {
			// TODO: see how this stands in practice
			// some resources have more than one action and we're de-duplicating them,
			// resulting in get-X + get-Y...
			// maybe something to check with scripts/spec_stats.py
			switch exec.Name() {
			case "create":
				create = exec
			case "get":
				read = exec
			case "update":
				update = exec
			case "delete":
				delete = exec
			case "list":
				list = exec
			default:
				connectionExecs = append(connectionExecs, exec)
			}
			return true, nil
		} else {
			return false, fmt.Errorf("Unsupported descriptor child type %T", child)
		}
	})
	if err != nil || create == nil {
		return resources, err
	}

	resourceName := strings.Join(path, "_")
	tflog.Debug(ctx, fmt.Sprintf("found resource %q", resourceName), debugMap)

	res, err := newMgcResource(ctx, sdk, resourceName, group.Description(), create, read, update, delete, list)
	if err != nil {
		tflog.Warn(ctx, err.Error(), debugMap)
		return resources, nil
	}

	tflog.Debug(ctx, fmt.Sprintf("export resource %q", resourceName), debugMap)
	resources = append(resources, func() resource.Resource { return res })

	for _, connectionCreate := range connectionExecs {
		path = append(path, connectionCreate.Name())
		name := strings.Join(path, "_")
		if strings.Contains(name, "get") {
			tflog.Debug(ctx, fmt.Sprintf("connection creation %s is a non-modifying action, it can't be turned into a resource", name))
			continue
		}

		connectionResource, err := newMgcConnectionResource(ctx, sdk, name, connectionCreate.Description(), connectionCreate, delete)
		if err != nil {
			tflog.Warn(ctx, err.Error(), debugMap)
			continue
		}

		resources = append(resources, func() resource.Resource { return connectionResource })
		tflog.Debug(ctx, fmt.Sprintf("export connection resource %q", resourceName), debugMap)
	}

	return resources, err
}

func (p *MgcProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return nil
}

func (p *MgcProvider) attrErr(resp *provider.ConfigureResponse, attr string, def ...string) {
	exArg := ""
	if len(def) > 0 {
		exArg = fmt.Sprintf("You can set this configuration use one of the following: set %s attribute", attr)
		exArg += strings.Join(def, ", ")
		exArg += "."
	}
	resp.Diagnostics.AddAttributeError(
		path.Root(attr),
		fmt.Sprintf("missing %s", attr),
		fmt.Sprintf("unable to create %s provider due to missing attributes.\n%s", providerTypeName, exArg),
	)
}

func New(version string, commit string, date string) func() provider.Provider {
	sdk := mgcSdk.NewSdk()

	return func() provider.Provider {
		return &MgcProvider{
			sdk:     sdk,
			version: version,
			commit:  commit,
			date:    date,
		}
	}
}
