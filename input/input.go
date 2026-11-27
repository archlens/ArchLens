package input

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"

	"github.com/archlens/archlens/utils"
	"github.com/bmatcuk/doublestar/v4"
)

type View struct {
	Include []string `json:"include"`
	Exclude []string `json:"exclude"`
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
		return true // the root, recursively, is everything
	}
	if ok, _ := doublestar.Match(pattern, dir); ok {
		return true
	}
	ok, _ := doublestar.Match(pattern+"/**", dir)
	return ok
}

// Match reports whether the package directory dir is included and not excluded.
func (v View) Match(dir string) bool {
	dir = path.Clean(filepath.ToSlash(dir))
	if !slices.ContainsFunc(v.Include, func(p string) bool { return matchDir(p, dir) }) {
		return false
	}
	return !slices.ContainsFunc(v.Exclude, func(p string) bool { return matchDir(p, dir) })
}

// func Match(dir string, views []View) bool {
// 	return slices.ContainsFunc(views, func(v View) bool { return v.Match(dir) })
// }

// // Keep returns the dependencies whose package is accepted by the set.
// func (s FilterSet) Keep(deps []parsers.Dependency) []parsers.Dependency {
// 	kept := make([]parsers.Dependency, 0, len(deps))
// 	for _, d := range deps {
// 		if s.Match(depDir(d)) {
// 			kept = append(kept, d)
// 		}
// 	}
// 	return kept
// }

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
		}
	}

	return input, nil
}

// func (v View) Filter() caching.Filter {
// 	return caching.Filter{Include: v.Include, Exclude: v.Exclude}
// }

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
