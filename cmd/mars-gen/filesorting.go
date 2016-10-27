package main

import (
	"go/ast"
	"sort"
)

type fInfo struct {
	Filename string
	File     *ast.File
}

type byName []fInfo

func (s byName) Len() int {
	return len(s)
}
func (s byName) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s byName) Less(i, j int) bool {
	return s[i].Filename < s[j].Filename
}

func getSortedFiles(pkg *ast.Package) []*ast.File {
	entries := make([]fInfo, 0, len(pkg.Files))
	for fn, f := range pkg.Files {
		entries = append(entries, fInfo{Filename: fn, File: f})
	}

	sort.Sort(byName(entries))

	slice := make([]*ast.File, len(entries))
	for idx, i := range entries {
		slice[idx] = i.File
	}

	return slice
}
