package openapi

import (
	"fmt"
	"regexp"
	"strings"

	"go.uber.org/zap"
	"golang.org/x/exp/slices"
	"magalu.cloud/core"
	"magalu.cloud/core/utils"

	"github.com/getkin/kin-openapi/openapi3"
)

type confirmation struct {
	Message string `json:"message"`
}

// Source -> Module -> Resource -> Operation

// Resource

func getServers(p *openapi3.PathItem, op *openapi3.Operation) openapi3.Servers {
	var servers openapi3.Servers
	if op.Servers != nil && len(*op.Servers) > 0 {
		servers = *op.Servers
	}
	if servers == nil && len(p.Servers) > 0 {
		servers = p.Servers
	}

	return servers
}

// NOTE: some OpenAPIs may have have similar operations with the same tag (== Resource),
// in order to disambiguate, we need to pass the whole set of names and then find the
// the action name, simplifying if no collisions
type operationDesc struct {
	path   *openapi3.PathItem
	op     *openapi3.Operation
	method string
	key    string
}

// a tree based on other maps or operationDesc
type operationTree struct {
	tree map[string]*operationTree
	desc *operationDesc
}

func (t *operationTree) Add(key []string, desc *operationDesc) error {
	if len(key) == 0 {
		t.desc = desc
		return nil
	}

	if t.tree == nil {
		t.tree = map[string]*operationTree{}
	}

	current := key[0]
	childT, ok := t.tree[current]
	if !ok {
		childT = &operationTree{}
		t.tree[current] = childT
	}

	return childT.Add(key[1:], desc)
}

type operationTreePath struct {
	key    string
	parent *operationTree
}

func (t *operationTree) VisitDesc(path []operationTreePath, visitor func(path []operationTreePath, desc *operationDesc) (bool, error)) (bool, error) {
	if t.desc != nil {
		if run, err := visitor(path, t.desc); !run || err != nil {
			return false, err
		}
	}

	for k, childT := range t.tree {
		oldLen := len(path)
		path = append(path, operationTreePath{k, t})

		if run, err := childT.VisitDesc(path, visitor); !run || err != nil {
			return false, err
		}

		path = path[:oldLen]
	}

	return true, nil
}

var openAPIPathArgRegex = regexp.MustCompile("[{](?P<name>[^}]+)[}]")

func getPathEntry(pathEntry string) (string, bool) {
	match := openAPIPathArgRegex.FindStringSubmatch(pathEntry)
	if len(match) > 0 {
		for i, substr := range match {
			if openAPIPathArgRegex.SubexpNames()[i] == "name" {
				return substr, true
			}
		}
	}

	return pathEntry, false
}

func getCoalescedPath(path []operationTreePath) []string {
	parts := []string{}
	wasVariable := false
	for i, p := range path {
		pathEntry, isVariable := getPathEntry(p.key)
		if i == 0 || len(p.parent.tree) > 1 || wasVariable {
			parts = append(parts, pathEntry)
		}
		wasVariable = isVariable
	}
	return parts
}

func getFullPath(path []operationTreePath) []string {
	parts := []string{}
	for _, p := range path {
		pathEntry, _ := getPathEntry(p.key)
		parts = append(parts, pathEntry)
	}
	return parts
}

func renamePath(httpMethod string, pathName string) string {
	switch httpMethod {
	case "post":
		return "create"
	case "put":
		return "replace"
	case "patch":
		return "update"
	case "get":
		// only consider "get" if ends with, mid-path are still list, ex:
		// GET:  /resource/{id}
		// LIST: /{containerId}/resource
		// GET:  /{containerId}/resource/{id}
		if strings.HasSuffix(pathName, "}") {
			return "get"
		}
		return "list"
	}

	return httpMethod
}

func getFullOperationName(httpMethod string, pathName string) []string {
	actionName := renamePath(httpMethod, pathName)
	name := []string{actionName}

	for _, pathEntry := range strings.Split(pathName, "/") {
		if pathEntry == "" {
			continue
		}
		name = append(name, pathEntry)
	}

	return name
}

func collectOperations(
	tag *openapi3.Tag,
	doc *openapi3.T,
	extensionPrefix *string,
	logger *zap.SugaredLogger,
) *operationTree {
	tree := &operationTree{}
	for key, path := range doc.Paths {
		if getHiddenExtension(extensionPrefix, path.Extensions) {
			continue
		}

		pathOps := map[string]*openapi3.Operation{
			"get":    path.Get,
			"post":   path.Post,
			"put":    path.Put,
			"patch":  path.Patch,
			"delete": path.Delete,
		}

		for method, op := range pathOps {
			if op == nil || getHiddenExtension(extensionPrefix, op.Extensions) {
				continue
			}

			if !slices.Contains(op.Tags, tag.Name) {
				continue
			}

			name := getFullOperationName(method, key)
			if err := tree.Add(name, &operationDesc{path, op, method, key}); err != nil {
				logger.Warnw("failed to add operation", "method", method, "key", key, "error", err)
			}
		}
	}

	return tree
}

func newResource(
	tag *openapi3.Tag,
	doc *openapi3.T,
	extensionPrefix *string,
	logger *zap.SugaredLogger,
	module *module,
) *core.SimpleGrouper[core.Executor] {
	logger = logger.Named(tag.Name)
	return core.NewSimpleGrouper[core.Executor](
		core.DescriptorSpec{
			Name:        getNameExtension(extensionPrefix, tag.Extensions, tag.Name),
			Description: getDescriptionExtension(extensionPrefix, tag.Extensions, tag.Description),
		},
		func() (operations []core.Executor, err error) {
			operations = []core.Executor{}
			operationsByName := map[string]core.Executor{}
			opTree := collectOperations(tag, doc, extensionPrefix, logger)

			_, err = opTree.VisitDesc([]operationTreePath{}, func(path []operationTreePath, desc *operationDesc) (bool, error) {
				opName := getNameExtension(extensionPrefix, desc.op.Extensions, "")
				if opName == "" {
					opName = strings.Join(getCoalescedPath(path), "-")
					if _, ok := operationsByName[opName]; ok {
						opName = strings.Join(getFullPath(path), "-")
					}
				}

				servers := getServers(desc.path, desc.op)
				if servers == nil {
					servers = doc.Servers
				}

				outputFlag, _ := getExtensionString(extensionPrefix, "output-flag", desc.op.Extensions, "")
				method := strings.ToUpper(desc.method)

				var operation core.Executor = newOperation(
					opName,
					desc,
					method,
					extensionPrefix,
					servers,
					logger,
					outputFlag,
					module,
				)

				isDelete := method == "DELETE"
				cExt, ok := getExtensionObject(extensionPrefix, "confirmable", desc.op.Extensions, nil)

				if (ok && cExt != nil) || isDelete {
					cExec, err := wrapInConfirmableExecutor(cExt, isDelete, operation)
					if err != nil {
						return false, err
					}
					operation = cExec
				}

				if wtExt, ok := getExtensionObject(extensionPrefix, "wait-termination", desc.op.Extensions, nil); ok && wtExt != nil {
					if tExec, err := wrapInTerminatorExecutor(operation, wtExt); err == nil {
						operation = tExec
					} else {
						return false, err
					}
				}

				err = module.execResolver.add(
					desc.op.OperationID,
					[]string{"paths", desc.key, desc.method},
					operation,
				)
				if err != nil {
					return false, err
				}
				operations = append(operations, operation)
				operationsByName[opName] = operation
				return true, nil
			})

			return operations, err
		},
	)
}

func wrapInConfirmableExecutor(cExt map[string]any, isDelete bool, exec core.Executor) (core.ConfirmableExecutor, error) {
	c := &confirmation{}

	if cExt != nil {
		if err := utils.DecodeValue(cExt, c); err != nil {
			return nil, fmt.Errorf("error decoding confirmable extension: %w", err)
		}
	}

	return core.NewConfirmableExecutor(exec, core.ConfirmPromptWithTemplate(c.Message)), nil
}
