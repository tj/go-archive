package archive

import (
	"archive/zip"
	"io"
	"os"
)

// Writer is the writer interface for archive formats.
type Writer interface {
	Open() error
	Close() error
	Add(os.FileInfo) (io.Writer, error)
}

// newZipWriter returns a new zip writer.
func newZipWriter(w io.Writer) *zipWriter {
	return &zipWriter{w: w}
}

// zipWriter is the zip implementation of archive.Writer.
type zipWriter struct {
	w   io.Writer
	zip *zip.Writer
}

// Open implementation.
func (w *zipWriter) Open() error {
	w.zip = zip.NewWriter(w.w)
	return nil
}

// Close implementation.
func (w *zipWriter) Close() error {
	return w.zip.Close()
}

// Add implementation.
func (w *zipWriter) Add(info os.FileInfo) (io.Writer, error) {
	h, err := zip.FileInfoHeader(info)
	if err != nil {
		return nil, err
	}

	h.Method = zip.Deflate

	return w.zip.CreateHeader(h)
}
