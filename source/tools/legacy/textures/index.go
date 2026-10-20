package textures

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var ErrNotFound = errors.New("texture not found")

type Result struct {
	Path      string
	Relative  string
	Ambiguous bool
}

type Index struct {
	root   string
	exact  map[string]string
	byName map[string][]string
	paths  []string
}

func NewIndex(root string) (*Index, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("texture root %q is not a directory", root)
	}
	idx := &Index{
		root:   root,
		exact:  make(map[string]string),
		byName: make(map[string][]string),
	}
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		switch ext {
		case ".bmp", ".png", ".jpg", ".jpeg", ".tga", ".dds":
		default:
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		key := strings.ToLower(rel)
		idx.exact[key] = rel
		base := strings.ToLower(filepath.Base(rel))
		idx.byName[base] = append(idx.byName[base], rel)
		idx.paths = append(idx.paths, rel)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(idx.paths)
	for name := range idx.byName {
		sort.Strings(idx.byName[name])
	}
	return idx, nil
}

func (i *Index) Resolve(name string) (Result, error) {
	if i == nil {
		return Result{}, errors.New("nil texture index")
	}
	norm := normalize(name)
	if norm == "" {
		return Result{}, ErrNotFound
	}
	if rel, ok := i.exact[strings.ToLower(norm)]; ok {
		return i.result(rel, false)
	}
	base := strings.ToLower(filepath.Base(norm))
	matches := i.byName[base]
	if len(matches) == 0 {
		return Result{}, fmt.Errorf("%w: %s", ErrNotFound, name)
	}
	rel := matches[0]
	return i.result(rel, len(matches) > 1)
}

func (i *Index) ResolveAll(name string) []Result {
	if i == nil {
		return nil
	}
	norm := normalize(name)
	if norm == "" {
		return nil
	}
	base := strings.ToLower(filepath.Base(norm))
	var out []Result
	for _, rel := range i.byName[base] {
		r, err := i.result(rel, false)
		if err == nil {
			out = append(out, r)
		}
	}
	return out
}

func (i *Index) Paths() []string {
	if i == nil {
		return nil
	}
	return append([]string(nil), i.paths...)
}

func (i *Index) result(rel string, ambiguous bool) (Result, error) {
	full := filepath.Join(i.root, filepath.FromSlash(rel))
	if _, err := os.Stat(full); err != nil {
		return Result{}, err
	}
	return Result{Path: full, Relative: rel, Ambiguous: ambiguous}, nil
}

func normalize(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "\\", "/")
	if len(name) >= 2 && name[1] == ':' {
		name = name[2:]
	}
	name = strings.TrimPrefix(name, "/")
	cleaned := strings.TrimSpace(name)
	for strings.Contains(cleaned, "//") {
		cleaned = strings.ReplaceAll(cleaned, "//", "/")
	}
	return cleaned
}
