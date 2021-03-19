package archive

import (
	"io"
	"os"
	"path/filepath"
	"sync/atomic"

	"github.com/apex/log"
	"github.com/dustin/go-humanize"
	"github.com/pkg/errors"
)

// Format is the kind of archive format.
type Format int

// Formats supported.
const (
	Zip Format = iota
)

// Transformer is the interface used to transform files.
type Transformer interface {
	// Transform a file or its meta-data. Note that the file info
	// is accepted as-is, so if you alter the reader contents
	// you must provide an appropriate .Size and so on.
	Transform(io.Reader, os.FileInfo) (io.Reader, os.FileInfo)
}

// TransformFunc implements the Transformer interface.
type TransformFunc func(io.Reader, os.FileInfo) (io.Reader, os.FileInfo)

// Transform implementation.
func (f TransformFunc) Transform(r io.Reader, i os.FileInfo) (io.Reader, os.FileInfo) {
	return f(r, i)
}

// Stats for an archive.
type Stats struct {
	FilesFiltered    int64
	DirsFiltered     int64
	FilesAdded       int64
	SizeUncompressed int64
}

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
	filter    Filter
	transform Transformer
	log       log.Interface
	w         Writer
	stats     Stats
}

// Stats returns stats about the archive.
func (a *Archive) Stats() *Stats {
	return &a.stats
}

// WithFilter adds a filter.
func (a *Archive) WithFilter(f Filter) *Archive {
	a.filter = f
	return a
}

// WithTransform adds a transform.
func (a *Archive) WithTransform(t Transformer) *Archive {
	a.transform = t
	return a
}

// Open the archive.
func (a *Archive) Open() error {
	a.log.Debug("open")
	return a.w.Open()
}

// AddDir directory recursively.
func (a *Archive) AddDir(root string) error {
	return filepath.Walk(root, func(abspath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		path, err := filepath.Rel(root, abspath)
		if err != nil {
			return err
		}
		path = filepath.Clean(path)

		if path == "." {
			return nil
		}

		info = &pathInfo{info, path}
		if a.filter != nil && a.filter.Match(info) {
			a.log.Debugf("filtered %s – %d", info.Name(), info.Size())

			if info.IsDir() {
				atomic.AddInt64(&a.stats.DirsFiltered, 1)
				return filepath.SkipDir
			}

			atomic.AddInt64(&a.stats.FilesFiltered, 1)
			return nil
		}

		if info.IsDir() {
			return nil
		}

		atomic.AddInt64(&a.stats.FilesAdded, 1)
		atomic.AddInt64(&a.stats.SizeUncompressed, info.Size())

		if info.Mode()&os.ModeSymlink != 0 {
			link, err := os.Readlink(info.Name())
			if err != nil {
				return errors.Wrap(err, "reading symlink")
			}

			w, err := a.Add(info)
			if err != nil {
				return errors.Wrap(err, "adding file")
			}

			if _, err := w.Write([]byte(link)); err != nil {
				return errors.Wrap(err, "writing link")
			}

			return nil
		}

		f, err := os.Open(abspath)
		if err != nil {
			return errors.Wrap(err, "opening file")
		}

		var r io.Reader = f
		if a.transform != nil {
			r, info = a.transform.Transform(r, info)
		}

		w, err := a.Add(info)
		if err != nil {
			return errors.Wrap(err, "adding file")
		}

		if _, err := io.Copy(w, r); err != nil {
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
		"files_filtered":    a.stats.FilesFiltered,
		"dirs_filtered":     a.stats.DirsFiltered,
		"files_added":       a.stats.FilesAdded,
		"size_uncompressed": humanize.Bytes(uint64(a.stats.SizeUncompressed)),
	}).Debug("stats")

	a.log.Debug("close")
	return a.w.Close()
}
