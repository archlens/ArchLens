package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/archlens/archlens/caching"
	"github.com/archlens/archlens/export"
	"github.com/archlens/archlens/graph"
	"github.com/archlens/archlens/utils/gitutils"

	"github.com/spf13/cobra"
)

// renderDiffCmd represents the renderDiff command
var renderDiffCmd = &cobra.Command{
	Use:   "render-diff [base-branch] <compare-branch>",
	Short: "Renders difference views comparing two git branches",
	Long: `Renders one diagram per view showing how package dependencies changed from
[base-branch] to <compare-branch>.

Links where the number of dependencies increased are green, links where it
decreased are red. Each link is labelled with the dependency count on
<compare-branch> followed by the change, e.g. "3 (+1)" or "0 (-2)".

Both branches are checked out into temporary git worktrees, so your working
tree is never touched. No cache is read or written.`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true // usage text is noise for runtime errors

		var baseRef, headRef string
		if len(args) == 2 {
			baseRef, headRef = args[0], args[1]
		} else {
			headRef = args[0]
		}

		configPath, _ := cmd.Flags().GetString("config")
		format, _ := cmd.Flags().GetString("format")
		noCache, _ := cmd.Flags().GetBool("no-cache")

		exporter, err := newExporter(format)
		if err != nil {
			return err
		}

		proj, err := loadProject(configPath)
		if err != nil {
			return err
		}

		if baseRef == "" {
			if proj.cfg.Github.Branch == "" {
				return fmt.Errorf("no base branch: set github.branch in archlens.json or pass [base-branch]")
			}
			baseRef = proj.cfg.Github.Branch
		}

		// The root folder may be a subdirectory of the repo; remember where.
		prefix, err := gitutils.Prefix(proj.absRoot)
		if err != nil {
			return fmt.Errorf("%s is not inside a git repository: %w", proj.absRoot, err)
		}

		// A nil RefCache means caching is off.
		var rc *caching.RefCache
		if !noCache {
			rc = &caching.RefCache{
				Dir:         filepath.Join(proj.cfgDir, ".archlens", "diff-cache"),
				Keep:        6,
				Fingerprint: caching.Fingerprint(proj.cfg.RunCommand, proj.filters()),
			}
		}

		baseGraphs, err := graphsAtRef(proj, prefix, baseRef, rc)
		if err != nil {
			return err
		}
		headGraphs, err := graphsAtRef(proj, prefix, headRef, rc)
		if err != nil {
			return err
		}

		for _, name := range proj.viewNames() {
			diff := graph.Compare(baseGraphs[name], headGraphs[name])
			title := fmt.Sprintf("%s (%s vs %s)", name, baseRef, headRef)
			output := exporter.ExportDiff(diff, title, proj.cfg.Name)
			path, err := export.Save(exporter, proj.cfg.SaveLocation, name+"_diff", output)
			if err != nil {
				return fmt.Errorf("saving diagram for %s: %w", name, err)
			}
			Sugar.Infof("Saved view %s to file %s\n", name+"_diff", path)

			var up, down int
			for _, e := range diff.Edges {
				if d := e.Delta(); d > 0 {
					up++
				} else if d < 0 {
					down++
				}
			}
			Sugar.Infof("%s: %d links increased, %d decreased", name, up, down)
		}
		return nil
	},
}

// graphsAtRef checks ref out into a temporary worktree, parses every view's files
// there and returns one graph per view. With rc == nil it uses a throwaway
// in-memory cache; nothing is loaded from or saved to disk.
func graphsAtRef(p *project, prefix, ref string, rc *caching.RefCache) (map[string]*graph.Graph, error) {
	wt, cleanup, err := gitutils.AddWorktree(p.absRoot, ref)
	if err != nil {
		return nil, fmt.Errorf("checking out %s: %w", ref, err)
	}
	defer cleanup()

	root := filepath.Join(wt, filepath.FromSlash(prefix))
	graphs := make(map[string]*graph.Graph, len(p.cfg.Views))

	if _, err := os.Stat(root); err != nil {
		// The root folder doesn't exist on this ref: treat it as empty.
		Sugar.Warnf("%s: root folder not present, treating as empty", ref)
		for name := range p.cfg.Views {
			graphs[name] = graph.NewGraph()
		}
		return graphs, nil
	}

	Sugar.Infof("parsing %s", ref)

	c, reparse := &caching.Cache{}, true
	if rc != nil {
		c, reparse = rc.Load(root, p.cfg.RunCommand), false
	}
	viewFiles, err := p.parse(c, root, reparse)
	if err != nil {
		return nil, fmt.Errorf("%w (on %s)", err, ref)
	}
	if rc != nil {
		if err := rc.Save(c); err != nil {
			Sugar.Warnf("saving cache for %s: %v", ref, err)
		}
	}

	for name, files := range viewFiles {
		graphs[name] = c.Graph(files, root, p.filters())
	}
	return graphs, nil
}

func init() {
	rootCmd.AddCommand(renderDiffCmd)
	renderDiffCmd.Flags().StringP("config", "c", "archlens.json", "Path to the archlens.json configuration file")
	renderDiffCmd.Flags().StringP("format", "f", "mermaid", formatHelp)
	renderDiffCmd.Flags().BoolP("no-cache", "n", false, "Runs the renderDiff without cache files thus allowing for full parsing")
}
