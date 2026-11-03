package parsers

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
)

// The parser sources are embedded into the binary at build time.
//
// go embed can only reach files inside this package's directory tree, and it
// refuses any directory that contains a go.mod (that is a different module).
// It would also be a problem to keep a `package main` ast_parser.go next to
// this file, since it would be compiled into package parsers. So the Go
// parser's files are stored with a .txt suffix and renamed when written out.

//go:embed ast_parsers/python/ast_parser.py
var pythonParser []byte

//go:embed ast_parsers/go/ast_parser.go.txt
var goParser []byte

//go:embed ast_parsers/go/go.mod.txt
var goMod []byte

//go:embed ast_parsers/java/AstParser.java
var javaParser []byte

type bundledFile struct {
	name string
	data []byte
}

// BundlePython writes the embedded Python parser (ast_parser.py) into location.
func BundlePython(location string) error {
	err := writeBundle(location, []bundledFile{
		{"ast_parser.py", pythonParser},
	})
	if err != nil {
		return fmt.Errorf("bundle python parser: %w", err)
	}
	return nil
}

// BundleGo writes the embedded Go parser (ast_parser.go) and its go.mod into location.
func BundleGo(location string) error {
	err := writeBundle(location, []bundledFile{
		{"ast_parser.go", goParser},
		{"go.mod", goMod},
	})
	if err != nil {
		return fmt.Errorf("bundle go parser: %w", err)
	}
	return nil
}

// BundlePython writes the embedded Python parser (ast_parser.py) into location.
func BundleJava(location string) error {
	err := writeBundle(location, []bundledFile{
		{"AstParser.java", javaParser},
	})
	if err != nil {
		return fmt.Errorf("bundle java parser: %w", err)
	}
	return nil
}


// writeBundle creates location if needed and writes every file into it,
// overwriting existing copies so the shipped parser always matches the binary.
func writeBundle(location string, files []bundledFile) error {
	if err := os.MkdirAll(location, 0o755); err != nil {
		return err
	}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(location, f.name), f.data, 0o644); err != nil {
			return err
		}
	}
	return nil
}
