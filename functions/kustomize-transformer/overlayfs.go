package main

import (
	"errors"
	"path/filepath"

	"sort"

	"sigs.k8s.io/kustomize/kyaml/filesys"
)

type OverlayFs struct {
	Lower filesys.FileSystem
	Upper filesys.FileSystem
}

// CleanedAbs implements filesys.FileSystem.
func (o *OverlayFs) CleanedAbs(path string) (filesys.ConfirmedDir, string, error) {
	d, name, err := o.Upper.CleanedAbs(path)
	if err == nil {
		return d, name, nil
	}
	return o.Lower.CleanedAbs(path)
}

// Create implements filesys.FileSystem.
func (o *OverlayFs) Create(path string) (filesys.File, error) {
	return o.Upper.Create(path)
}

// Exists implements filesys.FileSystem.
func (o *OverlayFs) Exists(path string) bool {
	r := o.Upper.Exists(path)
	if r {
		return r
	}
	return o.Lower.Exists(path)
}

// Glob implements filesys.FileSystem.
func (o *OverlayFs) Glob(pattern string) ([]string, error) {
	ru, erru := o.Upper.Glob(pattern)
	rl, errl := o.Lower.Glob(pattern)
	return uniqueStrings(append(ru, rl...)), errors.Join(erru, errl)
}

// IsDir implements filesys.FileSystem.
func (o *OverlayFs) IsDir(path string) bool {
	if o.Upper.Exists(path) {
		return o.Upper.IsDir(path)
	}
	return o.Lower.IsDir(path)
}

// Mkdir implements filesys.FileSystem.
func (o *OverlayFs) Mkdir(path string) error {
	err := o.Upper.Mkdir(path)
	if err == nil {
		return err
	}
	return o.Lower.Mkdir(path)
}

// MkdirAll implements filesys.FileSystem.
func (o *OverlayFs) MkdirAll(path string) error {
	err := o.Upper.MkdirAll(path)
	if err == nil {
		return err
	}
	return o.Lower.MkdirAll(path)
}

// Open implements filesys.FileSystem.
func (o *OverlayFs) Open(path string) (filesys.File, error) {
	r, err := o.Upper.Open(path)
	if err == nil {
		return r, err
	}
	return o.Lower.Open(path)
}

// ReadDir implements filesys.FileSystem.
func (o *OverlayFs) ReadDir(path string) ([]string, error) {
	ru, erru := o.Upper.ReadDir(path)
	rl, errl := o.Lower.ReadDir(path)
	return uniqueStrings(append(ru, rl...)), errors.Join(erru, errl)
}

// ReadFile implements filesys.FileSystem.
func (o *OverlayFs) ReadFile(path string) ([]byte, error) {
	r, err := o.Upper.ReadFile(path)
	if err == nil {
		return r, err
	}
	return o.Lower.ReadFile(path)
}

// RemoveAll implements filesys.FileSystem.
func (o *OverlayFs) RemoveAll(path string) error {
	return errors.Join(
		o.Upper.RemoveAll(path),
		o.Lower.RemoveAll(path),
	)
}

// Walk implements filesys.FileSystem.
func (o *OverlayFs) Walk(path string, walkFn filepath.WalkFunc) error {
	panic("not implemented")
}

// WriteFile implements filesys.FileSystem.
func (o *OverlayFs) WriteFile(path string, data []byte) error {
	err := o.Upper.WriteFile(path, data)
	if err == nil {
		return err
	}
	return o.Lower.WriteFile(path, data)
}

func uniqueStrings(input []string) []string {
	sort.Strings(input)
	unique := input[:0]
	for i, s := range input {
		if i == 0 || s != input[i-1] {
			unique = append(unique, s)
		}
	}
	return unique
}

var _ filesys.FileSystem = &OverlayFs{}
