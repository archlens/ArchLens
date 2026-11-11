package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// Dependency is one internal import. Go imports a whole package and the
// parser does not trace which file is used, so File is the package path.
type Dependency struct {
	Package string `json:"package"`
	File    string `json:"file"`
}

type Output struct {
	Package      string       `json:"package"`
	Dependencies []Dependency `json:"dependencies"`
}

func main() {
	var rootFlag string
	flag.StringVar(&rootFlag, "r", ".", "project root directory")
	flag.StringVar(&rootFlag, "root", ".", "project root directory")
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		exitWithError("error: missing file argument")
	}
	filePath := args[0]

	// 1. Resolve Root Path
	absRoot, err := filepath.Abs(rootFlag)
	if err != nil {
		exitWithError(fmt.Sprintf("error resolving root path: %v", err))
	}
	realRoot, err := filepath.EvalSymlinks(absRoot)
	if err != nil {
		exitWithError(fmt.Sprintf("error evaluating root symlinks: %v", err))
	}

	rootInfo, err := os.Stat(realRoot)
	if err != nil || !rootInfo.IsDir() {
		exitWithError("error: root must be an existing directory")
	}

	// 2. Resolve File Path
	// Spec: "resolved against working directory... parser must still behave correctly when run any other way"
	var absFile string
	if filepath.IsAbs(filePath) {
		absFile = filePath
	} else {
		// Try resolving relative to CWD first
		candidate, err := filepath.Abs(filePath)
		if err == nil {
			if _, statErr := os.Stat(candidate); statErr == nil {
				absFile = candidate
			}
		}
		// Fallback: resolve relative to provided root directory
		if absFile == "" {
			absFile = filepath.Join(realRoot, filePath)
		}
	}

	realFile, err := filepath.EvalSymlinks(absFile)
	if err != nil {
		exitWithError(fmt.Sprintf("error evaluating file symlinks: %v", err))
	}

	fileInfo, err := os.Stat(realFile)
	if err != nil || fileInfo.IsDir() {
		exitWithError(fmt.Sprintf("error reading file: %v", err))
	}

	// Verify file resides inside root
	relFileFromRoot, err := filepath.Rel(realRoot, realFile)
	if err != nil || strings.HasPrefix(relFileFromRoot, "..") || relFileFromRoot == ".." {
		exitWithError("error: file is outside project root")
	}

	// Verify file is not inside third-party vendor directories
	if isVendorOrThirdParty(relFileFromRoot) {
		exitWithError("error: parsed file is inside vendor or third-party directory")
	}

	// 3. Determine Package Path
	fileDir := filepath.Dir(realFile)
	relPkgDir, err := filepath.Rel(realRoot, fileDir)
	if err != nil {
		exitWithError(fmt.Sprintf("error deriving package path: %v", err))
	}
	currentPackage := normalizePackagePath(relPkgDir)

	// 4. Read module name from go.mod if present
	moduleName := getModuleName(realRoot)

	// 5. Parse AST
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, realFile, nil, parser.ImportsOnly)
	if err != nil {
		exitWithError(fmt.Sprintf("syntax error: %v", err))
	}

	// 6. Resolve Imports
	seenDeps := make(map[Dependency]bool)
	dependencies := []Dependency{} // never nil, so it encodes as [] and not null

	for _, imp := range node.Imports {
		if imp.Path == nil {
			continue
		}
		importPath := strings.Trim(imp.Path.Value, `"`)

		targetPkg, isInternal := resolveGoImport(importPath, moduleName, fileDir, realRoot)
		if !isInternal {
			continue
		}

		// Omit imports pointing to the parsed file's own package
		if targetPkg == currentPackage {
			continue
		}

		// The file is unknown for Go, so the package path stands in for it
		dep := Dependency{Package: targetPkg, File: targetPkg}
		if !seenDeps[dep] {
			seenDeps[dep] = true
			dependencies = append(dependencies, dep)
		}
	}

	// 7. Output Result
	output := Output{
		Package:      currentPackage,
		Dependencies: dependencies,
	}

	encoder := json.NewEncoder(os.Stdout)
	if err := encoder.Encode(output); err != nil {
		exitWithError(fmt.Sprintf("error generating output JSON: %v", err))
	}
}

func resolveGoImport(importPath, moduleName, fileDir, realRoot string) (string, bool) {
	var targetAbs string

	if moduleName != "" && strings.HasPrefix(importPath, moduleName) {
		// Module-absolute import (e.g., "github.com/org/repo/pkg/utils")
		relFromModule := strings.TrimPrefix(importPath, moduleName)
		relFromModule = strings.TrimPrefix(relFromModule, "/")
		targetAbs = filepath.Join(realRoot, filepath.FromSlash(relFromModule))
	} else if strings.HasPrefix(importPath, "./") || strings.HasPrefix(importPath, "../") {
		// Relative import (resolved relative to the file's directory)
		targetAbs = filepath.Join(fileDir, filepath.FromSlash(importPath))
	} else {
		// External standard library or third-party dependency
		return "", false
	}

	// Resolve symlinks
	realTarget, err := filepath.EvalSymlinks(targetAbs)
	if err != nil {
		return "", false
	}

	// Verify target is inside root
	targetRelFromRoot, err := filepath.Rel(realRoot, realTarget)
	if err != nil || strings.HasPrefix(targetRelFromRoot, "..") || targetRelFromRoot == ".." {
		return "", false
	}

	// Verify target exists and is a directory
	info, err := os.Stat(realTarget)
	if err != nil || !info.IsDir() {
		return "", false
	}

	// Omit vendor or third-party paths
	if isVendorOrThirdParty(targetRelFromRoot) {
		return "", false
	}

	return normalizePackagePath(targetRelFromRoot), true
}

func normalizePackagePath(p string) string {
	p = filepath.ToSlash(p)
	if p == "" || p == "." {
		return "."
	}
	return strings.TrimPrefix(p, "./")
}

func isVendorOrThirdParty(relPath string) bool {
	clean := filepath.ToSlash(relPath)
	parts := strings.Split(clean, "/")
	for _, part := range parts {
		if part == "vendor" || part == "node_modules" || part == ".venv" {
			return true
		}
	}
	return false
}

func getModuleName(realRoot string) string {
	goModPath := filepath.Join(realRoot, "go.mod")
	content, err := os.ReadFile(goModPath)
	if err != nil {
		return ""
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				return strings.Trim(fields[1], "`\"'")
			}
		}
	}
	return ""
}

func exitWithError(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}