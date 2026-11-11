//go:build local

package json

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func skipInCI(t *testing.T) {
	t.Helper()
	if os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") == "true" {
		t.Skip("skipping in CI")
	}
}

func loadDoc(t *testing.T, path string) document {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var d document
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&d); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	return d
}

func TestCompareJSONOutputs(t *testing.T) {
	skipInCI(t)

	oldDir, newDir := os.Getenv("OLD_JSON_DIR"), os.Getenv("NEW_JSON_DIR")
	if oldDir == "" || newDir == "" {
		t.Skip("OLD_JSON_DIR / NEW_JSON_DIR not set")
	}

	var opts = cmp.Options{
		cmpopts.SortSlices(func(a, b packageJSON) bool {
			if a.Name != b.Name {
				return a.Name < b.Name
			}
			return a.Path < b.Path
		}),
		cmpopts.SortSlices(func(a, b edgeJSON) bool {
			if a.FromPackage != b.FromPackage {
				return a.FromPackage < b.FromPackage
			}
			return a.ToPackage < b.ToPackage
		}),
		cmpopts.SortSlices(func(a, b relation) bool {
			if a.FromFile.Path != b.FromFile.Path {
				return a.FromFile.Path < b.FromFile.Path
			}
			return a.ToFile.Path < b.ToFile.Path
		}),
	}

	files, err := os.ReadDir(oldDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		t.Run(f.Name(), func(t *testing.T) {
			oldDoc := loadDoc(t, filepath.Join(oldDir, f.Name()))
			newDoc := loadDoc(t, filepath.Join(newDir, f.Name()))
			if diff := cmp.Diff(oldDoc, newDoc, opts); diff != "" {
				t.Errorf("outputs differ (-old +new):\n%s", diff)
			}
		})
	}
}
