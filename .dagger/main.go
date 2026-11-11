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
	"strings"

	"golang.org/x/sync/errgroup"
)

type ArchLensGo struct {
	Src *dagger.Directory
}

func New(
	// +defaultPath="./src"
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

// JSON OUTPUT TESTING DO NOT USE IN CI

// Repos analysed when none are passed in (GitHub "owner/name").
var defaultRepos = []string{
	"zeeguu/api",
}

// Python container with an (empty) venv named archlens, already on PATH.
func (m *ArchLensGo) OldArchLensEnv() *dagger.Container {
	return dag.Container().
		From("python:3.12-slim").
		WithExec([]string{"python", "-m", "venv", "/opt/archlens"}).
		WithEnvVariable("VIRTUAL_ENV", "/opt/archlens").
		WithEnvVariable("PATH", "/opt/archlens/bin:$PATH",
			dagger.ContainerWithEnvVariableOpts{Expand: true})
}

// Downloads each repo's tarball with curl into <owner>__<name>/.
func (m *ArchLensGo) FetchRepos(
	// +optional
	repos []string,
) *dagger.Directory {
	if len(repos) == 0 {
		repos = defaultRepos
	}
	base := dag.Container().
		From("alpine:3.20").
		WithExec([]string{"apk", "add", "--no-cache", "curl", "tar"})

	out := dag.Directory()
	for _, repo := range repos {
		name := strings.ReplaceAll(repo, "/", "__")
		dir := "/repos/" + name
		fetched := base.WithExec([]string{"sh", "-c", fmt.Sprintf(
			"mkdir -p %s && curl -fsSL https://codeload.github.com/%s/tar.gz/HEAD | tar -xz --strip-components=1 -C %s",
			dir, repo, dir,
		)})
		out = out.WithDirectory(name, fetched.Directory(dir))
	}
	return out
}

// Runs old and new ArchLens on every repo in parallel, then diffs the JSON.
func (m *ArchLensGo) TestJsonOutput(
	ctx context.Context,
	// +optional
	repos []string,
) (string, error) {
	repoDir := m.FetchRepos(repos)

	entries, err := repoDir.Entries(ctx)
	if err != nil {
		return "", err
	}
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = strings.TrimSuffix(e, "/")
	}

	oldCtr := m.OldArchLensEnv().WithDirectory("/repos", repoDir)
	newCtr := m.BuildEnv().WithDirectory("/repos", repoDir)

	oldFiles := make([]*dagger.File, len(names))
	newFiles := make([]*dagger.File, len(names))

	g, gctx := errgroup.WithContext(ctx)
	for i, name := range names {
		g.Go(func() error {
			f, err := oldCtr.
				WithWorkdir("/repos/" + name).
				// TODO: WithExec(old archlens cmd, writing /out/old.json)
				File("/out/old.json").Sync(gctx)
			oldFiles[i] = f
			return err
		})
		g.Go(func() error {
			f, err := newCtr.
				WithWorkdir("/repos/" + name).
				// TODO: WithExec(new archlens cmd, writing /out/new.json)
				File("/out/new.json").Sync(gctx)
			newFiles[i] = f
			return err
		})
	}
	if err := g.Wait(); err != nil {
		return "", err
	}

	oldOut, newOut := dag.Directory(), dag.Directory()
	for i, name := range names {
		oldOut = oldOut.WithFile(name+".json", oldFiles[i])
		newOut = newOut.WithFile(name+".json", newFiles[i])
	}

	return m.BuildEnv().
		WithDirectory("/json/old", oldOut).
		WithDirectory("/json/new", newOut).
		WithEnvVariable("OLD_JSON_DIR", "/json/old").
		WithEnvVariable("NEW_JSON_DIR", "/json/new").
		WithExec([]string{"go", "test", "-tags", "local", "-v",
			"-run", "TestCompareJSONOutputs", "./export/json/..."}).
		Stdout(ctx)
}
