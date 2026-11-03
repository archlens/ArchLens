package input

import (
	"encoding/json"
	"maps"
	"os"
	"slices"

	"github.com/archlens/ArchLens/utils"
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
	Views        map[string]View `json:"views"`
	SaveLocation string          `json:"saveLocation"`
	RunCommand   string          `json:"runCommand"`
}

var logger = utils.Sugar

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

func matchFiles(patterns []string) (map[string]struct{}, error) {
	files := make(map[string]struct{})
	for _, pattern := range patterns {
		matches, err := doublestar.FilepathGlob(pattern, doublestar.WithFailOnIOErrors())
		if err != nil {
			return files, err
		}
		for _, match := range matches {
			file, err := os.Open(match)
			if err != nil {
				return files, err
			}
			fileInfo, err := file.Stat()
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

func GetFiles(view *View, rootDir string) ([]string, error) {
	restore := utils.WithWorkingDirectory(rootDir)
	defer restore()

	include, err := matchFiles(view.Include)
	if err != nil {
		return []string{}, err
	}
	logger.Debugf("Included files: %v", include)
	exclude, err := matchFiles(view.Exclude)
	if err != nil {
		return []string{}, err
	}
	logger.Debugf("Excluded files: %v", exclude)

	for path := range exclude {
		delete(include, path)
	}
	return slices.Collect(maps.Keys(include)), nil
}
