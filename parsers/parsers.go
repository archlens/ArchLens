package parsers

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"slices"
	"strings"
	"sync"
	"unicode"
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
func GetASTs(files []string, rootDir string, runCommand string) []ASTResult {
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
				ast, err := GetAST(f, rootDir, runCommand)
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

func GetAST(filePath string, rootDir string, runCommand string) (*ASTNode, error) {
	parts, err := splitCommand(runCommand)
	if err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return nil, errors.New("runCommand is empty")
	}
	// program + its own arguments, then the parser flags from the spec
	args := slices.Concat(parts[1:], []string{"-r", rootDir, filePath})

	cmd := exec.Command(parts[0], args...)
	cmd.Dir = rootDir

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