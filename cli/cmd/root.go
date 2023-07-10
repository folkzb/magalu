package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/profusion/cobra-cloud/parser"
	"github.com/spf13/cobra"
)

type CmdBuildHandler struct {
	moduleLoaders   CmdLoaders
	resourceLoaders CmdLoaders
	actionLoaders   CmdLoaders
	loadActionFlags CmdLoader
}

type CmdLoader func()
type CmdLoaders map[string]CmdLoader

func safeOSArg(n int) string {
	if len(os.Args) <= n {
		return ""
	}

	return os.Args[n]
}

func RegexpMatchGroups(regex *regexp.Regexp, str string) map[string]string {
	match := regex.FindStringSubmatch(str)

	results := map[string]string{}
	for i, name := range match {
		results[regex.SubexpNames()[i]] = name
	}
	return results
}

var openAPIPathArgRegex = regexp.MustCompile("[{](?P<name>[^}]+)[}]")

func getActionName(action *parser.OpenAPIAction) string {
	if action == nil {
		return ""
	}

	name := []string{string(action.HttpMethod)}
	hasArgs := false

	for _, pathEntry := range strings.Split(action.PathName, "/") {
		match := RegexpMatchGroups(openAPIPathArgRegex, pathEntry)

		if len(match) != 0 {
			name = append(name, match["name"])
			hasArgs = true
		} else if hasArgs {
			name = append(name, pathEntry)
		}
	}

	return strings.Join(name, "-")
}

var openAPIFileNameRe = regexp.MustCompile("^(?P<name>[^.]+)(?:|[.]openapi)[.](?P<ext>json|yaml|yml)$")

func findAllOpenAPI() []*parser.OpenAPIFileInfo {
	cwd, err := os.Getwd()

	if err != nil {
		fmt.Print("Unable to get current working directory")
	}

	openAPIsDir := filepath.Join(cwd, openapisDir)
	items, err := os.ReadDir(openAPIsDir)

	if err != nil {
		fmt.Print("Unable to read OpenAPI files")
	}

	result := make([]*parser.OpenAPIFileInfo, 0)

	for _, item := range items {
		if item.IsDir() {
			continue
		}
		item.Info()

		matches := openAPIFileNameRe.FindStringSubmatch(item.Name())

		if len(matches) == 0 {
			continue
		}

		fileInfo := &parser.OpenAPIFileInfo{
			Name:      matches[1],
			Extension: matches[2],
			Path:      filepath.Join(openAPIsDir, item.Name()),
		}

		result = append(result, fileInfo)
	}

	return result
}

func loadParametersIntoCommand(action *parser.OpenAPIAction, cmd *cobra.Command) {
	action.PathParams, action.HeaderParam = GetParams(action.Parameters)
	action.RequestBodyParam = GetRequestBodyParams(action)

	for _, p := range action.PathParams {
		AddFlag(cmd, p)
	}
	for _, p := range action.HeaderParam {
		AddFlag(cmd, p)
	}
	for _, p := range action.RequestBodyParam {
		AddFlag(cmd, p)
	}
}

func createActionLoader(
	action *parser.OpenAPIAction,
	actionName string,
	parentCmd *cobra.Command,
	cmdBuildHandler *CmdBuildHandler,
) CmdLoader {
	load := func() {
		actionCmd := &cobra.Command{
			Use:   actionName,
			Short: action.Summary,
			Long:  action.Description,

			Run: func(cmd *cobra.Command, args []string) {
				// TODO: Actually execute action command here
				cmd.Help()
			},
		}

		cmdBuildHandler.loadActionFlags = func() {
			loadParametersIntoCommand(action, actionCmd)
		}

		nativeHelpFunc := actionCmd.HelpFunc()
		actionCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
			if !cmd.HasLocalFlags() {
				cmdBuildHandler.loadActionFlags()
			}
			nativeHelpFunc(cmd, args)
		})

		parentCmd.AddCommand(actionCmd)
	}

	return load
}

func createResourceLoader(
	module *parser.OpenAPIModule,
	tag *openapi3.Tag,
	parentCmd *cobra.Command,
	cmdBuildHandler *CmdBuildHandler,
) CmdLoader {
	load := func() {
		resourceCmd := &cobra.Command{
			Use:   tag.Name,
			Short: tag.Description,

			Run: func(cmd *cobra.Command, args []string) {
				cmd.Help()
			},
		}

		nativeHelpFunc := resourceCmd.HelpFunc()
		resourceCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
			for _, loadAction := range cmdBuildHandler.actionLoaders {
				loadAction()
			}
			nativeHelpFunc(cmd, args)
		})

		for _, action := range module.ActionsByTag()[tag] {
			actionName := getActionName(action)
			cmdBuildHandler.actionLoaders[actionName] = createActionLoader(action, actionName, resourceCmd, cmdBuildHandler)
		}

		parentCmd.AddCommand(resourceCmd)
	}

	return load
}

func createModuleLoader(
	openAPI *parser.OpenAPIFileInfo,
	parentCmd *cobra.Command,
	cmdBuildHandler *CmdBuildHandler,
) CmdLoader {
	load := func() {
		module, err := parser.LoadOpenAPI(openAPI)

		if err != nil {
			// TODO: Handle error
			return
		}

		moduleCmd := &cobra.Command{
			Use:     module.Name,
			Short:   module.Description,
			Version: module.Version,

			Run: func(cmd *cobra.Command, args []string) {
				cmd.Help()
			},
		}

		nativeHelpFunc := moduleCmd.HelpFunc()
		moduleCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
			for _, loadResource := range cmdBuildHandler.resourceLoaders {
				loadResource()
			}
			nativeHelpFunc(cmd, args)
		})

		sortedTags := module.Tags
		sort.Slice(sortedTags, func(i, j int) bool {
			return sortedTags[i].Name < sortedTags[j].Name
		})

		for _, tag := range sortedTags {
			cmdBuildHandler.resourceLoaders[tag.Name] = createResourceLoader(module, tag, moduleCmd, cmdBuildHandler)
		}

		parentCmd.AddCommand(moduleCmd)
	}

	return load
}

func createRootCmd(cmdBuildHandler *CmdBuildHandler) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cloud",
		Short: "CLI tool for OpenAPI integration",
		Long: `This CLI is a dynamic processor of OpenAPI files that
can generate a command line on-demand for Rest manipulation`,

		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	cmd.PersistentFlags().StringVar(&openapisDir, "openapis-dir", "openapis", "Input directory where OpenAPI files reside")

	nativeHelpFunc := cmd.HelpFunc()
	cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		for _, loadModule := range cmdBuildHandler.moduleLoaders {
			loadModule()
		}
		nativeHelpFunc(cmd, args)
	})

	return cmd
}

func initModuleLoaders(rootCmd *cobra.Command, cmdBuildHandler *CmdBuildHandler) {
	openAPIs := findAllOpenAPI()
	for _, openAPI := range openAPIs {
		cmdBuildHandler.moduleLoaders[openAPI.Name] = createModuleLoader(openAPI, rootCmd, cmdBuildHandler)
	}
}

func Execute() {
	cmdBuildHandler := &CmdBuildHandler{
		moduleLoaders:   make(CmdLoaders),
		resourceLoaders: make(CmdLoaders),
		actionLoaders:   make(CmdLoaders),
	}
	rootCmd := createRootCmd(cmdBuildHandler)

	currentArgIdx := 1
	if safeOSArg(currentArgIdx) == "help" {
		// Don't block loading of commands if native Help command (by Cobra) is called
		currentArgIdx += 1
	}

	initModuleLoaders(rootCmd, cmdBuildHandler)
	loadModule, canLoadModule := cmdBuildHandler.moduleLoaders[safeOSArg(currentArgIdx)]
	if !canLoadModule {
		fmt.Println("Invalid or missing module name!")
		rootCmd.Execute()
		return
	}
	currentArgIdx += 1

	loadModule()
	loadResource, canLoadResource := cmdBuildHandler.resourceLoaders[safeOSArg(currentArgIdx)]
	if !canLoadResource {
		fmt.Println("Invalid or missing resource name!")
		rootCmd.Execute()
		return
	}
	currentArgIdx += 1

	loadResource()
	loadAction, canLoadAction := cmdBuildHandler.actionLoaders[safeOSArg(currentArgIdx)]
	if !canLoadAction {
		fmt.Println("Invalid or missing action name!")
		rootCmd.Execute()
		return
	}

	loadAction()
	cmdBuildHandler.loadActionFlags()

	rootCmd.Execute()
}

var openapisDir string
