package input

import (
	"io/fs"
	"path"
	"path/filepath"
	"slices"
)

// Discover walks root once and returns, per view, the sorted files it needs:
// files matching the view's file patterns whose directory (their package) the
// view shows. It also returns the union over all views. Paths are relative to
// root with "/" separators. Directories every view excludes are not walked.
func Discover(root string, sels map[string]*Selector) (map[string][]string, map[string]struct{}, error) {
	viewFiles := make(map[string][]string, len(sels))
	union := make(map[string]struct{})

	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)

		if d.IsDir() {
			if rel == "." {
				return nil
			}
			if d.Name() == ".git" || d.Name() == CacheDir || excludedEverywhere(sels, rel) {
				return filepath.SkipDir
			}
			return nil
		}

		dir := path.Dir(rel) // the file's package; "." for the root
		for name, sel := range sels {
			if !sel.MatchFile(rel) {
				continue
			}
			if _, ok := sel.Node(dir); !ok {
				continue
			}
			viewFiles[name] = append(viewFiles[name], rel)
			union[rel] = struct{}{}
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	for _, files := range viewFiles {
		slices.Sort(files)
	}
	return viewFiles, union, nil
}

func excludedEverywhere(sels map[string]*Selector, dir string) bool {
	if len(sels) == 0 {
		return false
	}
	for _, s := range sels {
		if !s.Excluded(dir) {
			return false
		}
	}
	return true
}
