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

// GetASTs concurrently parses each file in `files` (relative to rootDir)
// and returns the collected results. Errors are captured per-file rather
// than aborting the whole batch.
func GetASTs(files []string, rootDir string, runCommand string) []ASTResult {
	numWorkers := runtime.NumCPU()
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
	parts := strings.Fields(runCommand)
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
		return nil, fmt.Errorf("bad json from python: %w", err)
	}
	return &root, nil
}