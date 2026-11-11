package cmd

import (
	"github.com/archlens/archlens/export"
	"github.com/archlens/archlens/export/html"
	"github.com/archlens/archlens/export/json"
	"github.com/archlens/archlens/export/mermaid"

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
		full, _ := cmd.Flags().GetBool("parse-full")
		format, _ := cmd.Flags().GetString("format")
		watch, _ := cmd.Flags().GetBool("watch")

		configPath := "archlens.json"
		if len(args) > 0 {
			configPath = args[0]
		}

		proj, err := loadProject(configPath)
		if err != nil {
			Sugar.Error(err)
			return
		}

		if watch {
			port, _ := cmd.Flags().GetInt("port")
			view, _ := cmd.Flags().GetString("view")
			if err := runWatch(cmd.Context(), proj, view, port, full); err != nil {
				Sugar.Error(err)
			}
			return
		}

		var exporter export.Exporter
		switch format {
		case "mermaid":
			exporter = mermaid.NewExporter(false)
		case "mermaid-md":
			exporter = mermaid.NewExporter(true)
		case "json":
			exporter = json.Exporter{}
		case "html":
			exporter = html.Exporter{Inner: mermaid.NewExporter(false)}
		default:
			Sugar.Errorf("'%s' is not a valid export format\n", format)
			return
		}

		viewFiles, err := proj.refresh(full)
		if err != nil {
			Sugar.Error(err)
			return
		}
		for name, files := range viewFiles {
			Sugar.Debugf("%s: %d files: %v", name, len(files), files)
			if len(files) == 0 {
				continue
			}
			g := proj.graph(files)
			output := exporter.ExportGraph(g, name, proj.cfg.Name)
			path, err := export.Save(exporter, proj.cfg.SaveLocation, name, output)
			if err != nil {
				Sugar.Errorf("Failed to save export: %v\n", err)
				continue
			}
			Sugar.Infof("%s: %d nodes, %d edges, location: %s", name, g.TotalNodes, g.TotalEdges, path)
		}
	},
}

func init() {
	rootCmd.AddCommand(renderCmd)
	renderCmd.Flags().BoolP("parse-full", "p", false, "A flag to fully parse any project again even if cache is present")
	renderCmd.Flags().StringP("format", "f", "mermaid", formatHelp)
	renderCmd.Flags().BoolP("watch", "w", false, "serve the diagram on localhost and reload on file changes")
	renderCmd.Flags().Int("port", 8080, "port used with --watch")
	renderCmd.Flags().String("view", "", "view to show with --watch (required if several are configured)")
}
