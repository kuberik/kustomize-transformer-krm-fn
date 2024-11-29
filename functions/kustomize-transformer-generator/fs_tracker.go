package main

import (
	"path/filepath"

	"sigs.k8s.io/kustomize/kyaml/filesys"
)

type FileSystemAccessTracker struct {
	fs            filesys.FileSystem
	accessedFiles map[string]interface{}
}

func NewFileSystemAccessTracker(fs filesys.FileSystem) *FileSystemAccessTracker {
	return &FileSystemAccessTracker{
		fs:            fs,
		accessedFiles: make(map[string]interface{}),
	}
}

// CleanedAbs implements filesys.FileSystem.
func (f *FileSystemAccessTracker) CleanedAbs(path string) (filesys.ConfirmedDir, string, error) {
	return f.fs.CleanedAbs(path)
}

// Create implements filesys.FileSystem.
func (f *FileSystemAccessTracker) Create(path string) (filesys.File, error) {
	return f.fs.Create(path)
}

// Exists implements filesys.FileSystem.
func (f *FileSystemAccessTracker) Exists(path string) bool {
	return f.fs.Exists(path)
}

// Glob implements filesys.FileSystem.
func (f *FileSystemAccessTracker) Glob(pattern string) ([]string, error) {
	return f.fs.Glob(pattern)
}

// IsDir implements filesys.FileSystem.
func (f *FileSystemAccessTracker) IsDir(path string) bool {
	return f.fs.IsDir(path)
}

// Mkdir implements filesys.FileSystem.
func (f *FileSystemAccessTracker) Mkdir(path string) error {
	return f.fs.Mkdir(path)
}

// MkdirAll implements filesys.FileSystem.
func (f *FileSystemAccessTracker) MkdirAll(path string) error {
	return f.fs.MkdirAll(path)
}

// Open implements filesys.FileSystem.
func (f *FileSystemAccessTracker) Open(path string) (filesys.File, error) {
	f.accessedFiles[path] = nil
	return f.fs.Open(path)
}

// ReadDir implements filesys.FileSystem.
func (f *FileSystemAccessTracker) ReadDir(path string) ([]string, error) {
	return f.fs.ReadDir(path)
}

// ReadFile implements filesys.FileSystem.
func (f *FileSystemAccessTracker) ReadFile(path string) ([]byte, error) {
	f.accessedFiles[path] = nil
	return f.fs.ReadFile(path)
}

// RemoveAll implements filesys.FileSystem.
func (f *FileSystemAccessTracker) RemoveAll(path string) error {
	return f.fs.RemoveAll(path)
}

// Walk implements filesys.FileSystem.
func (f *FileSystemAccessTracker) Walk(path string, walkFn filepath.WalkFunc) error {
	return f.fs.Walk(path, walkFn)
}

// WriteFile implements filesys.FileSystem.
func (f *FileSystemAccessTracker) WriteFile(path string, data []byte) error {
	return f.fs.WriteFile(path, data)
}

func (f *FileSystemAccessTracker) AccessedFiles() []string {
	var files []string
	for file := range f.accessedFiles {
		files = append(files, file)
	}
	return files
}

var _ filesys.FileSystem = &FileSystemAccessTracker{}
