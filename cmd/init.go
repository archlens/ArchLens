package cmd

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/archlens/archlens/input"
	"github.com/archlens/archlens/parsers"
	"github.com/archlens/archlens/utils/gitutils"
	"github.com/spf13/cobra"
)

// parserDir is the folder next to archlens.json that init bundles the parser
// into. runCommand runs from the directory of archlens.json, so the paths in it
// are relative to that, always with forward slashes so the file is portable.
const parserDir = "ast_parser"

// language describes how to set up one of the bundled parsers.
type language struct {
	runCommand string
	files      []string
	exclude    []string // package patterns
	bundle     func(location string) error
}

var languages = map[string]language{
	"python": {
		runCommand: "python3 " + path.Join(parserDir, "ast_parser.py"),
		files:      []string{"*.py", "!__init__.py"},
		exclude:    []string{"**/__pycache__", "venv", ".venv", "env", "**/site-packages"},
		bundle:     parsers.BundlePython,
	},
	"go": {
		runCommand: "go run " + path.Join(parserDir, "ast_parser.go"),
		files: []string{"*.go", "!*.pb.go", "!*.pb.gw.go", "!*_gen.go", "!*.gen.go",
			"!mock_*.go", "!*_mock.go"},
		exclude: []string{"**/vendor", "**/testdata"},
		bundle:  parsers.BundleGo,
	},
	"java": {
		runCommand: "java " + path.Join(parserDir, "AstParser.java"),
		files:      []string{"*.java", "!package-info.java", "!module-info.java"},
		exclude: []string{"**/target", "**/build", "**/out", "**/bin", "**/classes",
			"**/.gradle", "**/.mvn", "**/generated", "**/generated-sources"},
		bundle: parsers.BundleJava,
	},
	"csharp": {
		runCommand: path.Join(parserDir, "dist", csharpBinary()),
		files:      []string{"*.cs"},
		exclude:    []string{"**/bin", "**/obj", "**/packages", "**/.vs", "**/generated", "**/Generated"},
		bundle:     parsers.BundleCSharp,
	},
}

func csharpBinary() string {
	if runtime.GOOS == "windows" {
		return "csharp-parser.exe"
	}
	return "csharp-parser"
}

// newTemplate returns a fresh base config, so nothing is shared between calls.
func newTemplate() input.Input {
	return input.Input{
		Name:       "ProjectName",
		RootFolder: ".",
		RunCommand: "",
		Github: input.Github{
			Url:    "https://github.com/archlens/ArchLens",
			Branch: "master",
		},
		SaveLocation: "./diagrams/",
		Views: map[string]input.View{
			"completeView": {Include: []string{}, Exclude: []string{}},
		},
	}
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
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true // usage text is noise for runtime errors

		l, _ := cmd.Flags().GetString("location")

		var lang string
		if len(args) > 0 {
			lang = strings.ToLower(args[0])
		}

		configPath := "./archlens.json"
		if l != "" {
			configPath = l
		}

		info, err := os.Stat(configPath)
		switch {
		case err == nil && info.IsDir():
			// Existing directory: put archlens.json inside it.
			configPath = filepath.Join(configPath, "archlens.json")
		case filepath.Ext(configPath) == "":
			// No extension: treat it as a directory path.
			configPath = filepath.Join(configPath, "archlens.json")
		case !strings.EqualFold(filepath.Ext(configPath), ".json"):
			return fmt.Errorf("config path %q must be a .json file or a directory", configPath)
		}

		dir := filepath.Dir(configPath)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("could not create %s: %w", dir, err)
		}

		cfg := newTemplate()
		cfg.RootFolder = rootRelativeTo(dir)

		if lang != "" {
			if err := applyLanguage(&cfg, lang, filepath.Join(dir, parserDir)); err != nil {
				return err
			}
		}

		data, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			return fmt.Errorf("encoding archlens.json: %w", err)
		}
		if err := os.WriteFile(configPath, append(data, '\n'), 0o644); err != nil {
			return fmt.Errorf("could not write %s: %w", configPath, err)
		}
		Sugar.Infof("Wrote %s", configPath)

		// The caches live in .archlens/ next to the config and the root.
		repoRoot, err := gitutils.FindRoot(dir)
		if err != nil {
			Sugar.Warnf("Skipping .gitignore update: %s", err)
			return nil
		}
		changed, err := gitutils.EnsureCacheIgnored(repoRoot)
		if err != nil {
			Sugar.Warnf("Could not update .gitignore: %s", err)
			return nil
		}
		if changed {
			Sugar.Info("Added .archlens/ cache folder to .gitignore")
		}
		return nil
	},
}

// rootRelativeTo returns the current directory as a path relative to the config
// directory. rootFolder is resolved against the config's directory, so this keeps
// "the folder I ran init from" as the project root even when -l puts the config
// somewhere else.
func rootRelativeTo(cfgDir string) string {
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}
	absDir, err := filepath.Abs(cfgDir)
	if err != nil {
		return "."
	}
	rel, err := filepath.Rel(absDir, cwd)
	if err != nil {
		return "."
	}
	return filepath.ToSlash(rel)
}

// applyLanguage bundles the parser for name into location and points cfg at it.
func applyLanguage(cfg *input.Input, name, location string) error {
	lang, ok := languages[name]
	if !ok {
		return fmt.Errorf("language %q is not supported out of the box (supported: %s), "+
			"create your own AST-parser using the specification on the github repository",
			name, strings.Join(slices.Collect(maps.Keys(languages)), ", "))
	}
	if err := lang.bundle(location); err != nil {
		return fmt.Errorf("bundling the ast_parser at %s: %w", location, err)
	}

	cfg.RunCommand = lang.runCommand
	cfg.Files = lang.files
	view := cfg.Views["completeView"]
	view.Exclude = lang.exclude
	cfg.Views["completeView"] = view

	Sugar.Infof("Wrote parser to %s", location)
	return nil
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().StringP("location", "l", "", "Choose the location to put the default archlens.json file (defaults to ./archlens.json)")
}
