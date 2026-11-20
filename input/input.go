package input

import (
	"encoding/json"
	"os"
)

const CacheDir = ".archlens"

type View struct {
	Include    []string `json:"include"`
	Exclude    []string `json:"exclude"`
	Files      []string `json:"files,omitempty"`
	RunCommand string   `json:"runCommand,omitempty"`
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
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	in := &Input{}
	if err := json.Unmarshal(data, in); err != nil {
		return nil, err
	}
	return in, nil
}
