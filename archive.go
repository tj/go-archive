package archive

import (
	"io"
	"os"
	"path/filepath"
	"sync/atomic"

	"github.com/apex/log"
	humanize "github.com/dustin/go-humanize"
	"github.com/pkg/errors"
)

// Format is the kind of archive format.
type Format int

// Formats supported.
const (
	Zip Format = iota
)

// New returns a new archive writer.
func New(format Format, w io.Writer) *Archive {
	switch format {
	case Zip:
		return &Archive{
			log: log.Log,
			w:   newZipWriter(w),
		}
	default:
		panic("unsupported format")
	}
}

// NewZip returns a new zip archive.
func NewZip(w io.Writer) *Archive {
	return New(Zip, w)
}

// Archive wraps a format's writer to provide conveniences.
type Archive struct {
	filter Filter
	log    log.Interface
	w      Writer
	stats  struct {
		filteredFiles    int64
		filteredDirs     int64
		addedFiles       int64
		sizeUncompressed int64
	}
}

// WithFilter adds a filter.
func (a *Archive) WithFilter(f Filter) *Archive {
	a.filter = f
	return a
}

// Open the archive.
func (a *Archive) Open() error {
	a.log.Debug("open")
	return a.w.Open()
}

// AddDir directory recursively.
func (a *Archive) AddDir(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if path == "." {
			return nil
		}

		info = &pathInfo{info, path}
		if a.filter != nil && a.filter.Match(info) {
			a.log.Debugf("filtered %s – %d", info.Name(), info.Size())

			if info.IsDir() {
				atomic.AddInt64(&a.stats.filteredDirs, 1)
				return filepath.SkipDir
			}

			atomic.AddInt64(&a.stats.filteredFiles, 1)
			return nil
		}

		if info.IsDir() {
			return nil
		}

		if !info.Mode().IsRegular() {
			log.Debugf("skip irregular file %q", path)
			return nil
		}

		atomic.AddInt64(&a.stats.addedFiles, 1)
		atomic.AddInt64(&a.stats.sizeUncompressed, info.Size())

		w, err := a.Add(info)
		if err != nil {
			return errors.Wrap(err, "adding file")
		}

		f, err := os.Open(path)
		if err != nil {
			return errors.Wrap(err, "opening file")
		}

		if _, err := io.Copy(w, f); err != nil {
			return errors.Wrap(err, "copying file")
		}

		if err := f.Close(); err != nil {
			return errors.Wrap(err, "closing file")
		}

		return nil
	})
}

// Add a file.
func (a *Archive) Add(info os.FileInfo) (io.Writer, error) {
	a.log.Debugf("add %s: size=%d mode=%s", info.Name(), info.Size(), info.Mode())
	return a.w.Add(info)
}

// Close the archive.
func (a *Archive) Close() error {
	a.log.WithFields(log.Fields{
		"filtered_files":    a.stats.filteredFiles,
		"filtered_dirs":     a.stats.filteredDirs,
		"files_added":       a.stats.addedFiles,
		"size_uncompressed": humanize.Bytes(uint64(a.stats.sizeUncompressed)),
	}).Debug("stats")

	a.log.Debug("close")
	return a.w.Close()
}
