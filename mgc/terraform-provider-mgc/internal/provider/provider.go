package provider

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	mgcSdk "magalu.cloud/sdk"
)

var _ provider.Provider = (*MgcProvider)(nil)

type MgcProvider struct {
	version string
	commit  string
	date    string
	sdk     *mgcSdk.Sdk
}

type MgcProviderModel struct {
	ApiToken types.String   `tfsdk:"api_token"`
	Apis     []types.String `tfsdk:"apis"`
}

func (p *MgcProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	tflog.Debug(ctx, "setting provider metadata")
	resp.TypeName = "magalu"
	resp.Version = p.version
}

func (p *MgcProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	tflog.Debug(ctx, "setting provider schema")
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"api_token": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Token used to authenticate with the MagaluID platform.",
			},
			"apis": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Which MagaluCloud products need to be supported by the provider.",
				// Should we use validators to know if the apis exist? Or return an error later
				// Validators: ,
			},
		},
	}
}

func (p *MgcProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Debug(ctx, "configuring MGC provider")
	var config MgcProviderModel

	// Load all configurations
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)

	apiToken := config.ApiToken.ValueString()
	if apiToken == "" {
		p.attrErr(resp, "access_token", "set MGC_TF_API_TOKEN environment variable")
		return
	}

	apis := strings.Split(os.Getenv("MGC_TF_PRODUCT_APIS"), ",")
	if config.Apis != nil && len(config.Apis) != 0 {
		apis = make([]string, len(config.Apis))
		for i, v := range config.Apis {
			apis[i] = v.ValueString()
		}
		tflog.Debug(ctx, "using `apis` property from tf file")
	}
	if len(apis) == 0 {
		p.attrErr(resp, "apis", "set MGC_TF_PRODUCT_APIS environment variable")
		return
	}

	p.sdk.Auth().ApiToken = apiToken

	resp.DataSourceData = p.sdk
	resp.ResourceData = p.sdk
}

func (p *MgcProvider) Resources(ctx context.Context) []func() resource.Resource {
	return nil
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
		fmt.Sprintf("unable to create MagaluCloud provider due to missing attributes.\n%s", exArg),
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
