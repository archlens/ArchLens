package input

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"

	"github.com/archlens/archlens/parsers"
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
			Exclude: append(view.Exclude, ".archlens/*"),
		}
	}

	return input, nil
}

func (v View) Filter() parsers.Filter {
	return parsers.Filter{Include: v.Include, Exclude: v.Exclude}
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

	filter := view.Filter()
	var files []string
	for f := range matched {
		f = filepath.ToSlash(f)
		if filter.Match(path.Dir(f)) {
			files = append(files, f)
		}
	}
	slices.Sort(files)
	utils.Sugar.Debugf("Files: %v", files)
	return files, nil
}
