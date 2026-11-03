package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/archlens/ArchLens/input"
	"github.com/archlens/ArchLens/parsers"
	"github.com/archlens/ArchLens/utils/gitutils"
	"github.com/spf13/cobra"
)

var template input.Input = input.Input{
	Name:       "ProjectName",
	RootFolder: ".",
	RunCommand: "",
	Github: input.Github{
		Url:    "https://github.com/archlens/ArchLens",
		Branch: "master",
	},
	SaveLocation: "./diagrams/",
	Views: map[string]input.View{
		"completeView": input.View{
			Include: []string{},
			Exclude: []string{},
		},
	},
}

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init [language]",
	Short: "Initializes the project with a base archlens.json file and optionally [language] based ast_parser",
	Long: `Initializes the project with a base archlens.json file as well as an embedded ast_parser in the specified optional [language] argument.
	The project currently ships with python, java, and go ast_parsers. Users can create their own parsers by using the specification that can be found in the docs folder
	on the github page. Archlens communicates with the parser using a unified json struct through IPC so it's easy to build your own. 
	
	The location for the archlens.json file as well as the ast_parsers can be specified using the -l flag the parser folder is always initialized next to
	the archlens.json file.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		l, _ := cmd.Flags().GetString("location")

		var language string
		if len(args) > 0 {
			language = args[0]
		}

		path := "./archlens.json"
		if l != "" {
			path = l
		}

		info, err := os.Stat(path)
		switch {
		case err == nil && info.IsDir():
			// Existing directory: put archlens.json inside it.
			path = filepath.Join(path, "archlens.json")
		case filepath.Ext(path) == "":
			// No extension: treat it as a directory path.
			path = filepath.Join(path, "archlens.json")
		case !strings.EqualFold(filepath.Ext(path), ".json"):
			Sugar.Errorf("config path %q must be a .json file or a directory", path)
			return
		}

		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			Sugar.Errorf("Could not create %s: %s", dir, err)
			return
		}

		if language != "" {
			pickLanguage(language, filepath.Join(dir, "ast_parser"))
		}

		data, err := json.MarshalIndent(template, "", "  ")
		if err != nil {
			panic(err)
		}
		data = append(data, '\n')

		err = os.WriteFile(path, data, 0644)
		if err != nil {
			Sugar.Errorf("Could not write archlens.json: %s", err.Error())
			return
		}
		Sugar.Info("Wrote archlens.json to current location")

		root, err := gitutils.FindRoot(".")
		if err != nil {
			Sugar.Warnf("Skipping .gitignore update: %s", err.Error())
			return
		}

		changed, err := gitutils.EnsureCacheIgnored(root)
		if err != nil {
			Sugar.Warnf("Could not update .gitignore: %s", err.Error())
			return
		}

		if changed {
			Sugar.Info("Added .archlens/ cache folder to .gitignore")
		}
	},
}

func pickLanguage(language, location string) {
	language = strings.ToLower(language)
	var err error
	var path string
	completeView := template.Views["completeView"]

	switch language {
	case "python":
		path = filepath.Join(location, "ast_parser.py")
		template.RunCommand = "python3 " + path
		completeView.Include = []string{"**/*.py", "*.py"}
		completeView.Exclude = []string{"__pycache__", "**/__pycache__/**", "**/__init__.py", "__init__.py",
			"venv", ".venv", "env", ".env", "virtualenv", "site-packages", "**/site-packages/**"}
		err = parsers.BundlePython(location)
	case "go":
		path = filepath.Join(location, "ast_parser.go")
		template.RunCommand = "go run " + path
		completeView.Include = []string{"**/*.go", "*.go"}
		completeView.Exclude = []string{"vendor", "**/vendor/**", "testdata", "**/testdata/**", "*.pb.go", "*.pb.gw.go", "*_gen.go", "*.gen.go", "mock_*.go", "*_mock.go", "go.mod", "go.sum"}
		err = parsers.BundleGo(location)
	case "java":
		path = filepath.Join(location, "AstParser.java")
		template.RunCommand = "java " + path
		completeView.Include = []string{"**/*.java", "*.java"}
		completeView.Exclude = []string{"target", "build", "out", "bin", "classes",
			".gradle", ".mvn", "generated", "generated-sources", "**/generated/**",
			"**/generated-sources/**", "**/build/generated/**", "package-info.java", "**/package-info.java",
			"module-info.java", "**/module-info.java"}
		err = parsers.BundleJava(location)
	default:
		Sugar.Errorf("Language is not supported out of the box yet please create your own AST-parser using the specification on the github repository")
	}
	if err != nil {
		Sugar.Errorf("Problem bundling the ast_parser at location: %s\nwith error: %s", location, err)
	} else {
		Sugar.Infof("Wrote parser to %s", path)
	}
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().StringP("location", "l", "", "Choose the location to put the default archlens.json file (defaults to ./archlens.json)")
}
