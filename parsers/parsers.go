package parsers

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"unicode"

	"github.com/bmatcuk/doublestar/v4"
)

// Dependency is one internal import reported by a parser. File is the
// imported file, or the package path when the parser can't tell the file
// (see the AST spec).
type Dependency struct {
	Package string `json:"package"`
	File    string `json:"file"`
}

type ASTNode struct {
	Package      string       `json:"package"`
	Dependencies []Dependency `json:"dependencies"`
}

type ASTResult struct {
	File string
	AST  *ASTNode
	Err  error
}

// Filter selects packages by directory (relative to the project root).
// Patterns are globs and match recursively: "cmd" matches cmd and cmd/x/y.
type Filter struct {
	Include []string `json:"include"`
	Exclude []string `json:"exclude"`
}

// FilterSet accepts a package if any of its filters does.
type FilterSet []Filter

func matchDir(pattern, dir string) bool {
	pattern = path.Clean(filepath.ToSlash(pattern))
	if pattern == "." {
		return true // the root, recursively, is everything
	}
	if ok, _ := doublestar.Match(pattern, dir); ok {
		return true
	}
	ok, _ := doublestar.Match(pattern+"/**", dir)
	return ok
}

// Match reports whether the package directory dir is included and not excluded.
func (f Filter) Match(dir string) bool {
	dir = path.Clean(filepath.ToSlash(dir))
	if !slices.ContainsFunc(f.Include, func(p string) bool { return matchDir(p, dir) }) {
		return false
	}
	return !slices.ContainsFunc(f.Exclude, func(p string) bool { return matchDir(p, dir) })
}

func (s FilterSet) Match(dir string) bool {
	return slices.ContainsFunc(s, func(f Filter) bool { return f.Match(dir) })
}

// depDir is the directory of the package a dependency points to.
// Mirrors graph.AddFile: File == Package means File is the package path.
func depDir(d Dependency) string {
	if d.File == d.Package {
		return path.Clean(filepath.ToSlash(d.File))
	}
	return path.Dir(filepath.ToSlash(d.File))
}

// Keep returns the dependencies whose package is accepted by the set.
func (s FilterSet) Keep(deps []Dependency) []Dependency {
	kept := make([]Dependency, 0, len(deps))
	for _, d := range deps {
		if s.Match(depDir(d)) {
			kept = append(kept, d)
		}
	}
	return kept
}

// splitCommand splits a command line such as `python3 "my dir/arch.py" --fast`
// into its arguments.
//
// Whitespace separates arguments. Single or double quotes group text,
// whitespace included, into one argument and are removed; a quote may also
// start mid-word (--name="two words" becomes --name=two words). An empty pair
// of quotes produces an empty argument.
//
// Backslashes are NOT escape characters, so Windows paths such as
// C:\tools\parser.exe pass through unchanged. An unterminated quote is an error.
func splitCommand(s string) ([]string, error) {
	var (
		args  []string
		cur   strings.Builder
		inArg bool // the current argument has started, even if it is still empty ("")
		quote rune // the active quote character, 0 outside quotes
	)

	for _, r := range s {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				cur.WriteRune(r)
			}
		case r == '"' || r == '\'':
			quote = r
			inArg = true
		case unicode.IsSpace(r):
			if inArg {
				args = append(args, cur.String())
				cur.Reset()
				inArg = false
			}
		default:
			cur.WriteRune(r)
			inArg = true
		}
	}

	if quote != 0 {
		return nil, fmt.Errorf("unterminated %c quote in command %q", quote, s)
	}
	if inArg {
		args = append(args, cur.String())
	}
	return args, nil
}

// workerCount returns how many workers to start for the given number of jobs:
// one per CPU, but never more workers than there are jobs.
func workerCount(jobs int) int {
	return min(runtime.NumCPU(), jobs)
}

// GetASTs concurrently parses each file in `files` (relative to rootDir)
// and returns the collected results. Errors are captured per-file rather
// than aborting the whole batch.
func GetASTs(files []string, rootDir, cfgDir string, runCommand string, filters FilterSet) []ASTResult {
	numWorkers := workerCount(len(files))
	jobs := make(chan string, len(files))
	resultsCh := make(chan ASTResult, len(files))

	// 1. Enqueue jobs
	for _, f := range files {
		jobs <- f
	}
	close(jobs)

	// 2. Spawn a fixed pool of workers
	var wg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for f := range jobs {
				ast, err := GetAST(f, rootDir, cfgDir, runCommand, filters)
				resultsCh <- ASTResult{File: f, AST: ast, Err: err}
			}
		}()
	}

	// 3. Wait for workers to finish
	wg.Wait()
	close(resultsCh)

	// 4. Collect results
	results := make([]ASTResult, 0, len(files))
	for r := range resultsCh {
		results = append(results, r)
	}
	return results
}

func GetAST(filePath, rootDir, cfgDir, runCommand string, filters FilterSet) (*ASTNode, error) {
	parts, err := splitCommand(runCommand)
	if err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return nil, errors.New("runCommand is empty")
	}

	// The parser runs in cfgDir, so it needs an absolute path to find the file.
	absFile := filepath.Join(rootDir, filepath.FromSlash(filePath))
	args := slices.Concat(parts[1:], []string{"-r", rootDir, absFile})

	cmd := exec.Command(parts[0], args...)
	cmd.Dir = cfgDir

	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("command failed: %s", exitErr.Stderr)
		}
		return nil, err
	}

	var root ASTNode
	if err := json.Unmarshal(out, &root); err != nil {
		return nil, fmt.Errorf("bad json from parser: %w", err)
	}
	return &root, nil
}
