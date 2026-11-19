package cmd

import (
	"github.com/archlens/archlens/export"

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
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true // usage text is noise for runtime errors

		full, _ := cmd.Flags().GetBool("parse-full")
		format, _ := cmd.Flags().GetString("format")
		watch, _ := cmd.Flags().GetBool("watch")

		configPath := "archlens.json"
		if len(args) > 0 {
			configPath = args[0]
		}

		proj, err := loadProject(configPath)
		if err != nil {
			return err
		}

		if watch {
			if cmd.Flags().Changed("format") {
				Sugar.Warnf("--format is ignored with --watch, the live view is always html")
			}
			port, _ := cmd.Flags().GetInt("port")
			view, _ := cmd.Flags().GetString("view")
			return runWatch(cmd.Context(), proj, view, port, full)
		}

		exporter, err := newExporter(format)
		if err != nil {
			return err
		}

		viewFiles, err := proj.refresh(full)
		if err != nil {
			return err
		}

		for _, name := range proj.viewNames() { // sorted, so logs come out in a stable order
			files := viewFiles[name]
			Sugar.Debugf("%s: %d files: %v", name, len(files), files)
			if len(files) == 0 {
				continue
			}
			g := proj.graph(files)
			output := exporter.ExportGraph(g, name, proj.cfg.Name)
			path, err := export.Save(exporter, proj.cfg.SaveLocation, name, output)
			if err != nil {
				Sugar.Errorf("Failed to save export for %s: %v", name, err)
				continue
			}
			Sugar.Infof("%s: %d nodes, %d edges, location: %s", name, g.TotalNodes, g.TotalEdges, path)
		}
		return nil
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
