package input

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/archlens/archlens/utils"
	"github.com/bmatcuk/doublestar/v4"
)

type View struct {
	Include []string `json:"include"`
	Exclude []string `json:"exclude"`
	Merge   *bool    `json:"merge,omitempty"`
}

type Github struct {
	Url    string `json:"url"`
	Branch string `json:"branch"`
}

type Input struct {
	Name         string          `json:"name"`
	Github       Github          `json:"github"`
	RootFolder   string          `json:"rootFolder"`
	Files        []string        `json:"files"`
	Views        map[string]View `json:"views"`
	SaveLocation string          `json:"saveLocation"`
	RunCommand   string          `json:"runCommand"`
}

func matchDir(pattern, dir string) bool {
	pattern = path.Clean(filepath.ToSlash(pattern))
	if pattern == "." {
		return true
	}
	if ok, _ := doublestar.Match(pattern, dir); ok {
		return true
	}
	ok, _ := doublestar.Match(pattern+"/**", dir)
	return ok
}

// Match reports whether the package path is included and not excluded.
func (v View) Match(dir string) bool {
	dir = path.Clean(filepath.ToSlash(dir))
	if !slices.ContainsFunc(v.Include, func(p string) bool { return matchDir(p, dir) }) {
		return false
	}
	return !slices.ContainsFunc(v.Exclude, func(p string) bool { return matchDir(p, dir) })
}

func (v View) MergeSubpackages() bool { return v.Merge == nil || *v.Merge }

func Load(path string) (*Input, error) {
	input := &Input{}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, input)
	if err != nil {
		return nil, err
	}
	// Add exlusion pattern for cache directory
	for name, view := range input.Views {
		input.Views[name] = View{
			Include: view.Include,
			Exclude: append(view.Exclude, ".archlens"),
			Merge: view.Merge,
		}
	}

	return input, nil
}

func matchFiles(patterns []string) (map[string]struct{}, error) {
	files := make(map[string]struct{})
	for _, pattern := range patterns {
		matches, err := doublestar.FilepathGlob(pattern, doublestar.WithFailOnIOErrors())
		if err != nil {
			return files, err
		}
		for _, match := range matches {
			fileInfo, err := os.Stat(match)
			if err != nil {
				return files, err
			}
			if !fileInfo.IsDir() {
				files[match] = struct{}{}
			}
		}
	}
	return files, nil
}

// GetFiles returns the source files of the packages selected by the view:
// <include>/**/<ext> for every include pattern and extension, minus files in
// excluded packages. Paths are relative to rootDir, with forward slashes.
func GetFiles(view *View, exts []string, rootDir string) ([]string, error) {
	if len(exts) == 0 {
		return nil, fmt.Errorf(`"files" is empty, so there is no file extension to look for`)
	}
	restore := utils.WithWorkingDirectory(rootDir)
	defer restore()

	var patterns []string
	for _, inc := range view.Include {
		for _, ext := range exts {
			patterns = append(patterns, path.Join(filepath.ToSlash(inc), "**", ext))
		}
	}
	matched, err := matchFiles(patterns)
	if err != nil {
		return nil, err
	}

	// filter := view.Filter()
	var files []string
	for f := range matched {
		f = filepath.ToSlash(f)
		if view.Match(f) {
			files = append(files, f)
		}
	}
	slices.Sort(files)
	utils.Sugar.Debugf("Files: %v", files)
	return files, nil
}

// Root returns the directory that a package living in dir is merged into: the
// shortest leading part of dir that one of the include patterns matches on its
// own. With include ["cmd"], "cmd/sub/x" gives "cmd". With ["internal/*"],
// "internal/a/b" gives "internal/a". The bool is false when nothing should be
// merged: dir is unknown or the root, or only "." matches it.
func (v View) Root(dir string) (string, bool) {
	dir = path.Clean(filepath.ToSlash(dir))
	if dir == "." {
		return "", false
	}
	segments := strings.Split(dir, "/")
	for i := 1; i <= len(segments); i++ {
		prefix := strings.Join(segments[:i], "/")
		for _, p := range v.Include {
			p = path.Clean(filepath.ToSlash(p))
			if p == "." {
				continue // the whole tree: keep full detail
			}
			if ok, _ := doublestar.Match(p, prefix); ok {
				return prefix, true
			}
		}
	}
	return "", false
}
