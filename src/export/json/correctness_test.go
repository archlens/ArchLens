//go:build local

package json

import (
	"bytes"
	"encoding/json"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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

var opts = cmp.Options{
	cmpopts.IgnoreFields(document{}, "Title"),
	cmpopts.IgnoreFields(packageJSON{}, "Path"),
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

func normName(s string) string { return strings.ReplaceAll(s, ".", "/") }

// normalizeOld maps the old exporter's format onto the new one.
// root is the absolute project root the old tool ran in, with a trailing "/".
func normalizeOld(d document, root string) document {
	rel := func(f fileRef) fileRef {
		return fileRef{Name: f.Name, Path: strings.TrimPrefix(f.Path, root)}
	}
	out := document{Title: d.Title}
	for _, p := range d.Packages {
		p.Name = normName(p.Name)
		out.Packages = append(out.Packages, p)
	}
	for _, e := range d.Edges {
		e.FromPackage = normName(e.FromPackage)
		e.ToPackage = normName(e.ToPackage)
		rels := make([]relation, len(e.Relations))
		for i, r := range e.Relations {
			rels[i] = relation{FromFile: rel(r.FromFile), ToFile: rel(r.ToFile)}
		}
		e.Relations = rels
		out.Edges = append(out.Edges, e)
	}
	return out
}

// restrictTo drops packages (and edges touching them) that ref doesn't have.
func restrictTo(d, ref document) document {
	known := map[string]bool{}
	for _, p := range ref.Packages {
		known[p.Name] = true
	}
	out := document{Title: d.Title}
	for _, p := range d.Packages {
		if known[p.Name] {
			out.Packages = append(out.Packages, p)
		}
	}
	for _, e := range d.Edges {
		if known[e.FromPackage] && known[e.ToPackage] {
			out.Edges = append(out.Edges, e)
		}
	}
	return out
}

func envBool(key string) bool {
	v, err := strconv.ParseBool(os.Getenv(key))
	return err == nil && v
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// LOGGING FOR BETTER OUTPUT FORMATTING

type testWriter struct{ t *testing.T }

func (w testWriter) Write(p []byte) (int, error) {
	w.t.Log(strings.TrimRight(string(p), "\n"))
	return len(p), nil
}

func newTestLogger(t *testing.T) *zap.SugaredLogger {
	cfg := zap.NewDevelopmentEncoderConfig()
	cfg.TimeKey = ""
	cfg.CallerKey = ""
	cfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(cfg),
		zapcore.AddSync(testWriter{t}),
		zapcore.DebugLevel,
	)
	return zap.New(core).Sugar()
}

func setDiff(a, b []string) (onlyA, onlyB []string) {
	inA, inB := map[string]bool{}, map[string]bool{}
	for _, s := range a {
		inA[s] = true
	}
	for _, s := range b {
		inB[s] = true
	}
	for s := range inA {
		if !inB[s] {
			onlyA = append(onlyA, s)
		}
	}
	for s := range inB {
		if !inA[s] {
			onlyB = append(onlyB, s)
		}
	}
	sort.Strings(onlyA)
	sort.Strings(onlyB)
	return
}

func pkgNames(d document) []string {
	out := make([]string, len(d.Packages))
	for i, p := range d.Packages {
		out[i] = p.Name
	}
	return out
}

func edgeMap(d document) map[string]edgeJSON {
	m := make(map[string]edgeJSON, len(d.Edges))
	for _, e := range d.Edges {
		m[e.FromPackage+" -> "+e.ToPackage] = e
	}
	return m
}

func relKeys(e edgeJSON) []string {
	out := make([]string, len(e.Relations))
	for i, r := range e.Relations {
		out[i] = r.FromFile.Path + " -> " + r.ToFile.Path
	}
	return out
}

func keys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// report logs a readable summary and returns how many differences it found.
func report(log *zap.SugaredLogger, oldDoc, newDoc document) int {
	n := 0

	missing, extra := setDiff(pkgNames(oldDoc), pkgNames(newDoc))
	for _, p := range missing {
		log.Errorf("package only in OLD: %s", p)
		n++
	}
	for _, p := range extra {
		log.Warnf("package only in NEW: %s", p)
		n++
	}

	oldE, newE := edgeMap(oldDoc), edgeMap(newDoc)
	missing, extra = setDiff(keys(oldE), keys(newE))
	for _, k := range missing {
		log.Errorf("edge only in OLD: %s", k)
		n++
	}
	for _, k := range extra {
		log.Warnf("edge only in NEW: %s", k)
		n++
	}

	var shared []string
	for k := range oldE {
		if _, ok := newE[k]; ok {
			shared = append(shared, k)
		}
	}
	sort.Strings(shared)
	for _, k := range shared {
		o, nw := oldE[k], newE[k]
		if o.Label != nw.Label {
			log.Errorf("edge %s: label old=%s new=%s", k, o.Label, nw.Label)
			n++
		}
		if o.State != nw.State {
			log.Errorf("edge %s: state old=%s new=%s", k, o.State, nw.State)
			n++
		}
		onlyOld, onlyNew := setDiff(relKeys(o), relKeys(nw))
		for _, r := range onlyOld {
			log.Errorf("edge %s: relation only in OLD: %s", k, r)
			n++
		}
		for _, r := range onlyNew {
			log.Warnf("edge %s: relation only in NEW: %s", k, r)
			n++
		}
	}
	return n
}

//END LOGGING

func TestCompareJSONOutputs(t *testing.T) {
	skipInCI(t)

	log := newTestLogger(t)

	oldP := envOr("OLD_JSON", "old.json")
	newP := envOr("NEW_JSON", "new.json")
	root := os.Getenv("ROOT") // project root the old tool ran in
	if root != "" {
		root = strings.TrimSuffix(root, "/") + "/"
	}
	strict := envBool("STRICT_JSON")
	t.Logf("strict=%v old=%s new=%s", strict, oldP, newP)

	oldDoc := normalizeOld(loadDoc(t, oldP), root)
	newDoc := loadDoc(t, newP)
	if !strict {
		newDoc = restrictTo(newDoc, oldDoc)
	}

	if !cmp.Equal(oldDoc, newDoc, opts) {
		if report(log, oldDoc, newDoc) == 0 {
			t.Log(cmp.Diff(oldDoc, newDoc, opts)) // summary missed it, show raw diff
		}
		t.Fail()
	}
}
