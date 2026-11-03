package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/archlens/ArchLens/caching"
	"github.com/archlens/ArchLens/export"
	"github.com/archlens/ArchLens/export/json"
	"github.com/archlens/ArchLens/export/mermaid"
	"github.com/archlens/ArchLens/utils/gitutils"
	"github.com/archlens/ArchLens/graph"
	"github.com/archlens/ArchLens/input"
	
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
		} else if len(args) == 1 {
			headRef = args[0]
		} else {
			Sugar.Panicf("Too few args need minimum 1 arg: compare-branch")
		}

		configPath, _ := cmd.Flags().GetString("config")
		format, _ := cmd.Flags().GetString("format")

		var exporter export.Exporter
		switch format {
		case "mermaid":
			exporter = mermaid.NewExporter(false)
		case "mermaid-md":
			exporter = mermaid.NewExporter(true)
		case "json":
			exporter = json.Exporter{}
		default:
			Sugar.Panicf("'%s' is not a valid export format\n", format)
		}
		noCache, _ := cmd.Flags().GetBool("no-cache")

		cfg, err := input.Load(configPath)
		if err != nil {
			return fmt.Errorf("loading configuration: %w", err)
		}

		if baseRef == "" && cfg.Github.Branch == "" {
			Sugar.Panicf("No baseRef defined in archlens.json and no base-branch given as input argument")
		} else if baseRef == "" {
			baseRef = cfg.Github.Branch
		}

		absRoot, err := filepath.Abs(cfg.RootFolder)
		if err != nil {
			return fmt.Errorf("resolving root folder: %w", err)
		}
		// The root folder may be a subdirectory of the repo; remember where.
		prefix, err := gitutils.Prefix(absRoot)
		if err != nil {
			return fmt.Errorf("%s is not inside a git repository: %w", absRoot, err)
		}

		// The cache lives next to archlens.json, so running from a subfolder
		// still finds the same cache. A nil RefCache means caching is off.
		var rc *caching.RefCache
		if !noCache {
			absCfg, err := filepath.Abs(configPath)
			if err != nil {
				return fmt.Errorf("resolving config path: %w", err)
			}
			rc = &caching.RefCache{
				Dir:         filepath.Join(filepath.Dir(absCfg), ".archlens", "diff-cache"),
				Keep:        6,
				Fingerprint: caching.Fingerprint(cfg.RunCommand),
			}
		}

		baseGraphs, err := graphsAtRef(absRoot, prefix, baseRef, cfg, rc)
		if err != nil {
			return err
		}
		headGraphs, err := graphsAtRef(absRoot, prefix, headRef, cfg, rc)
		if err != nil {
			return err
		}

		for name := range cfg.Views {
			diff := graph.Compare(baseGraphs[name], headGraphs[name])
			title := fmt.Sprintf("%s (%s vs %s)", name, baseRef, headRef)
			output := exporter.ExportDiff(diff, title)
			path, err := export.Save(exporter, cfg.SaveLocation, name+"_diff", output)
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
// there and returns one graph per view. It uses a throwaway in-memory cache purely
// for its parse-and-build logic; nothing is loaded from or saved to disk.
func graphsAtRef(repoDir, prefix, ref string, cfg *input.Input, rc *caching.RefCache) (map[string]*graph.Graph, error) {
	wt, cleanup, err := gitutils.AddWorktree(repoDir, ref)
	if err != nil {
		return nil, fmt.Errorf("checking out %s: %w", ref, err)
	}
	defer cleanup()

	root := filepath.Join(wt, filepath.FromSlash(prefix))
	graphs := make(map[string]*graph.Graph, len(cfg.Views))

	if _, err := os.Stat(root); err != nil {
		// The root folder doesn't exist on this ref: treat it as empty.
		Sugar.Warnf("%s: root folder not present, treating as empty", ref)
		for name := range cfg.Views {
			graphs[name] = graph.NewGraph()
		}
		return graphs, nil
	}

	viewFiles := make(map[string][]string, len(cfg.Views))
	union := make(map[string]struct{})
	for name, view := range cfg.Views {
		files, err := input.GetFiles(&view, root)
		if err != nil {
			return nil, fmt.Errorf("getting files for %s on %s: %w", name, ref, err)
		}
		for i, f := range files {
			files[i] = filepath.ToSlash(filepath.Clean(f))
			union[files[i]] = struct{}{}
		}
		viewFiles[name] = files
	}

	Sugar.Infof("parsing %s", ref)

	c := &caching.Cache{}
	reparse := true
	if rc != nil {
		c, reparse = rc.Load(root, cfg.RunCommand), false
	}
	if err := c.Refresh(root, union, cfg.RunCommand, reparse); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", ref, err)
	}
	if rc != nil {
		if err := rc.Save(c); err != nil {
			Sugar.Warnf("saving cache for %s: %v", ref, err)
		}
	}
	for name, files := range viewFiles {
		graphs[name] = c.Graph(files, repoDir)
	}
	return graphs, nil
}

func init() {
	rootCmd.AddCommand(renderDiffCmd)
	renderDiffCmd.Flags().StringP("config", "c", "archlens.json", "Path to the archlens.json configuration file")
	renderDiffCmd.Flags().StringP("format", "f", "mermaid", "Set the output format for archlens [mermaid|mermaid-md|json]")
	renderDiffCmd.Flags().BoolP("no-cache", "n", false, "Runs the renderDiff without cache files thus allowing for full parsing")
}
