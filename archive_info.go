package archive

import (
	"os"
	"time"
)

// pathInfo wraps FileInfo to support a full path
// in place of Name(), which just makes the API
// a little simpler.
type pathInfo struct {
	os.FileInfo
	path string
}

// Name returns the full path.
func (p *pathInfo) Name() string {
	return p.path
}

// FileInfo implements the os.FileInfo interface,
// and is useful if you're working with
// in-memory contents instead of files
// from disk.
type FileInfo struct {
	Info
}

// Name implementation.
func (i *FileInfo) Name() string {
	return i.Info.Name
}

// Size implementation.
func (i *FileInfo) Size() int64 {
	return i.Info.Size
}

// Mode implementation.
func (i *FileInfo) Mode() os.FileMode {
	return i.Info.Mode
}

// ModTime implementation.
func (i *FileInfo) ModTime() time.Time {
	return i.Info.Modified
}

// IsDir implementation.
func (i *FileInfo) IsDir() bool {
	return i.Info.Dir
}

// Sys implementation.
func (i *FileInfo) Sys() interface{} {
	return nil
}

// Info implements the os.FileInfo interface,
// and is useful if you're working with
// in-memory contents instead of files
// from disk.
type Info struct {
	Name     string
	Size     int64
	Mode     os.FileMode
	Modified time.Time
	Dir      bool
}

// FileInfo returns the info wrapped as an os.FileInfo.
func (i Info) FileInfo() os.FileInfo {
	return &FileInfo{i}
}
