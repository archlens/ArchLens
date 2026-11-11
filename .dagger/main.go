// A generated module for ArchLensGo functions
//
// This module has been generated via dagger init and serves as a reference to
// basic module structure as you get started with Dagger.
//
// Two functions have been pre-created. You can modify, delete, or add to them,
// as needed. They demonstrate usage of arguments and return types using simple
// echo and grep commands. The functions can be called from the dagger CLI or
// from one of the SDKs.
//
// The first line in this comment block is a short description line and the
// rest is a long description with more detail on the module's purpose or usage,
// if appropriate. All modules should have a short description.

package main

import (
	"context"
	"dagger/arch-lens-go/internal/dagger"
	"fmt"
)

type ArchLensGo struct {
	Src *dagger.Directory
}

func New(
	// +defaultPath="."
	// +ignore=[".github", ".gitignore", ".git", ".archlens", ".vscode", ".dagger", "docs", "diagrams", "ast_parsers", "*.md"]
	src *dagger.Directory,
) *ArchLensGo {
	return &ArchLensGo{
		Src: src,
	}
}

func (m *ArchLensGo) WithSource(src *dagger.Directory) *ArchLensGo {
	m.Src = src
	return m
}

func (m *ArchLensGo) BuildEnv() *dagger.Container {
	return dag.Container().
		From("golang:1.26-bookworm").
		WithDirectory("./src", m.Src).
		WithWorkdir("./src").WithExec([]string{"go", "mod", "tidy"})
}

func (m *ArchLensGo) Build() *dagger.Directory {

	// define build matrix
	gooses := []string{"linux", "darwin", "windows"}
	goarches := []string{"amd64", "arm64"}

	// create empty directory to put build artifacts
	outputs := dag.Directory()

	golang := m.BuildEnv()


	for _, goos := range gooses {
		for _, goarch := range goarches {
			// create directory for each OS and architecture
			target := fmt.Sprintf("%s-%s", goos, goarch)
			buildDir := "build/" + target + "/"

			bin := "archlens"
				if goos == "windows" {
					bin += ".exe"
			}

			// build artifact
			build := golang.
				WithEnvVariable("GOOS", goos).
				WithEnvVariable("GOARCH", goarch).
				WithExec([]string{"go", "build", "-o", buildDir + bin})

			// add build to outputs
			outputs = outputs.WithDirectory(target, build.Directory(buildDir))
		}
	}

	return outputs
}

// +check
func (m *ArchLensGo) Test(ctx context.Context) (string, error) {
	return m.BuildEnv().
		WithExec([]string{"go", "test", "./..."}).
		Stdout(ctx)
}

// +check
func (m *ArchLensGo) Lint(ctx context.Context) (string, error) {
	return dag.Container().From("golangci/golangci-lint:latest-alpine").
		WithDirectory("./src", m.Src).
		WithWorkdir("./src").
		WithExec([]string{"golangci-lint", "run"}).Stdout(ctx)
}
