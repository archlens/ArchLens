package cmd

import (
	"path/filepath"

	"github.com/archlens/ArchLens/caching"
	"github.com/archlens/ArchLens/export"
	"github.com/archlens/ArchLens/export/mermaid"
	"github.com/archlens/ArchLens/export/json"

	"github.com/archlens/ArchLens/input"
	"github.com/spf13/cobra"
)

// renderCmd represents the render command
var renderCmd = &cobra.Command{
	Use:   "render",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		// probably move this and make it a bit more generalized
		p, _ := cmd.Flags().GetBool("parse-full")
		markdown, _ := cmd.Flags().GetBool("markdown")
		format, _ := cmd.Flags().GetString("format")

		var configPath string
		if len(args) == 0 {
			configPath = "archlens.json"
		} else {
			configPath = args[0]
		}

		var exporter export.Exporter
		switch format {
		case "mermaid":
			exporter = mermaid.NewExporter(markdown)
		case "json":
			exporter = json.Exporter{}
			// Sugar.Errorln("json export is not implemented yet")
		default:
			Sugar.Errorf("'%s' is not a valid export format\n", format)
			return
		}

		res, err := input.Load(configPath)
		if err != nil {
			Sugar.Errorf("Error when trying to load configuration: %v", err)
			return
		}
		Sugar.Infof("Config: %+v", *res)

		absRoot, err := filepath.Abs(res.RootFolder)
		if err != nil {
			Sugar.Errorf("Error resolving root folder: %v", err)
			return
		}
		cachePath := filepath.Join(absRoot, ".archlens", "cache.json")
		cache := caching.LoadCache(cachePath)

		viewFiles := make(map[string][]string)
		union := make(map[string]struct{})
		for name, view := range res.Views {
			files, err := input.GetFiles(&view, res.RootFolder)
			if err != nil {
				Sugar.Errorf("Error when trying to get files for %s: %v", name, err)
				continue
			}
			for i, f := range files {
				files[i] = filepath.ToSlash(filepath.Clean(f))
				union[files[i]] = struct{}{}
			}
			viewFiles[name] = files
		}

		if err := cache.Refresh(absRoot, union, res.RunCommand, p); err != nil {
			Sugar.Errorf("Error refreshing cache: %v", err)
			return
		}

		// Graphs are cheap to build from cached results, so rebuild one per view.
		for name, files := range viewFiles {
			Sugar.Debugf("%s: %d files: %v", name, len(files), files)
			if len(files) == 0 {
				continue
			}
			g := cache.Graph(files, absRoot)
			output := exporter.ExportGraph(g, name)
			path, err := export.Save(exporter, res.SaveLocation, name, output)
			if err != nil {
				Sugar.Errorf("Failed to save export: %v\n", err)
			}
			Sugar.Infof("%s: %d nodes, %d edges, location: %s", name, g.TotalNodes, g.TotalEdges, path)
		}

		if err := cache.Save(cachePath); err != nil {
			Sugar.Errorf("Error saving cache: %v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(renderCmd)
	renderCmd.Flags().BoolP("parse-full", "p", false, "A flag to fully parse any project again even if cache is present")
	renderCmd.Flags().Bool("markdown", false, "Export diagrams in markdown files")
	renderCmd.Flags().StringP("format", "f", "mermaid", "Set the output format for archlens [mermaid|json]. ")
}
