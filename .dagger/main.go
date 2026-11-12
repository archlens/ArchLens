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
	"strconv"
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

type repoSpec struct{ name, repo, ref string }

func parseRepo(s string) repoSpec {
	repo, ref, _ := strings.Cut(s, "@")
	if ref == "" {
		ref = "HEAD"
	}
	return repoSpec{name: strings.ReplaceAll(repo, "/", "__"), repo: repo, ref: ref}
}

func fetchRepo(spec repoSpec) *dagger.Directory {
	dir := "/repo"
	return dag.Container().
		From("alpine:3.20").
		WithExec([]string{"apk", "add", "--no-cache", "curl", "tar"}).
		WithExec([]string{"sh", "-c", fmt.Sprintf(
			"mkdir -p %s && curl -fsSL https://codeload.github.com/%s/tar.gz/%s | tar -xz --strip-components=1 -C %s",
			dir, spec.repo, spec.ref, dir,
		)}).
		Directory(dir)
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

// New ArchLens built from the current source and put on PATH.
func (m *ArchLensGo) newArchLensEnv() *dagger.Container {
	return m.BuildEnv().
		WithExec([]string{"go", "build", "-o", "/usr/local/bin/archlens", "."}).
		WithExec([]string{"./archlens", "init", "python"})
}

// TODO(you): ctr has the repo mounted and workdir set to the repo root.
// Install/run the OLD ArchLens here and return the JSON it produced.
func oldJSON(ctr *dagger.Container) *dagger.File {
	return ctr.
		WithExec([]string{"mkdir", "-p", "/out"}).
		// WithExec([]string{...}).
		File("/out/old.json")
}

// TODO(you): same for the NEW ArchLens (binary is already on PATH).
func newJSON(ctr *dagger.Container) *dagger.File {
	return ctr.
		WithExec([]string{"mkdir", "-p", "/out"}).
		WithExec([]string{"./archlens", "render"}).
		File("/out/new.json")
}

// Runs old and new ArchLens on each repo in parallel, then compares the JSON.
func (m *ArchLensGo) TestJsonOutput(
	ctx context.Context,
	// +optional
	repos []string,
	// Require exact equality instead of ignoring packages the old output lacks
	// +optional
	strict bool,
) (string, error) {
	if len(repos) == 0 {
		repos = defaultRepos
	}

	oldEnv := m.OldArchLensEnv()
	newEnv := m.newArchLensEnv()

	outputs := make([]string, len(repos))
	failed := make([]bool, len(repos))

	g, gctx := errgroup.WithContext(ctx)
	for i, r := range repos {
		g.Go(func() error {
			spec := parseRepo(r)
			src := fetchRepo(spec)
			root := "/repos/" + spec.name

			oldF := oldJSON(oldEnv.WithDirectory(root, src).WithWorkdir(root))
			newF := newJSON(newEnv.WithDirectory(root, src).WithWorkdir(root))

			// Return type "any" so a failing test doesn't abort the other repos.
			ctr := m.BuildEnv().
				WithFile("/work/old.json", oldF).
				WithFile("/work/new.json", newF).
				WithEnvVariable("OLD_JSON", "/work/old.json").
				WithEnvVariable("NEW_JSON", "/work/new.json").
				WithEnvVariable("ROOT", root).
				WithEnvVariable("STRICT_JSON", strconv.FormatBool(strict)).
				WithExec([]string{"go", "test", "-tags", "local", "-v", "-count=1",
					"-run", "TestCompareJSONOutputs", "./export/json/"},
					dagger.ContainerWithExecOpts{Expect: dagger.ReturnTypeAny})

			code, err := ctr.ExitCode(gctx)
			if err != nil {
				return err
			}
			stdout, err := ctr.Stdout(gctx)
			if err != nil {
				return err
			}
			stderr, err := ctr.Stderr(gctx)
			if err != nil {
				return err
			}
			outputs[i] = stdout + stderr
			failed[i] = code != 0
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return "", err
	}

	var report strings.Builder
	anyFailed := false
	for i, r := range repos {
		status := "PASS"
		if failed[i] {
			status, anyFailed = "FAIL", true
		}
		fmt.Fprintf(&report, "===== %s: %s =====\n%s\n", r, status, outputs[i])
	}
	if anyFailed {
		return "", fmt.Errorf("JSON outputs differ:\n%s", report.String())
	}
	return report.String(), nil
}
