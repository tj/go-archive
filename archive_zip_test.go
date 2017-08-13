package archive_test

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/tj/assert"
	"github.com/tj/go-archive"
)

func init() {
	// log.SetLevel(log.DebugLevel)
}

// tree helper.
func tree(dir string) (out string, err error) {
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if dir == path {
			return nil
		}

		path = strings.Replace(path, dir+"/", "", 1)

		if info.IsDir() {
			out += fmt.Sprintf("%s mode=%s\n", path, info.Mode())
		} else {
			out += fmt.Sprintf("%s mode=%s size=%d\n", path, info.Mode(), info.Size())
		}

		return nil
	})

	return
}

// TODO: shared tests that each implementation utilizes

// unzip reader to a tmp dir which is returned.
func unzip(t testing.TB, r io.Reader) (dir string) {
	// TODO: convert to archive/zip

	out, err := ioutil.TempDir(os.TempDir(), "up-")
	assert.NoError(t, err, "tmpdir")

	dst := filepath.Join(out, "out.zip")

	f, err := os.Create(dst)
	assert.NoError(t, err, "create")

	_, err = io.Copy(f, r)
	assert.NoError(t, err, "copy")

	err = f.Close()
	assert.NoError(t, err, "close")

	cmd := exec.Command("unzip", "out.zip")
	cmd.Dir = out
	assert.NoError(t, cmd.Run(), "unzip")

	err = os.Remove(dst)
	assert.NoError(t, err, "remove")

	return out
}

func TestZipWriter_files(t *testing.T) {
	var buf bytes.Buffer
	zip := archive.NewZip(&buf)
	assert.NoError(t, zip.Open(), "open")

	for i := 0; i < 10; i++ {
		s := strings.Repeat("Hello", i+1)

		a, err := zip.Add(archive.Info{
			Name: fmt.Sprintf("example-%d.txt", i),
			Size: int64(len(s)),
		}.FileInfo())

		assert.NoError(t, err, "add")

		_, err = io.WriteString(a, s)
		assert.NoError(t, err, "write")
	}

	assert.NoError(t, zip.Close(), "open")

	dir := unzip(t, &buf)
	files, err := ioutil.ReadDir(dir)
	assert.NoError(t, err, "readdir")

	for i, file := range files {
		name := fmt.Sprintf("example-%d.txt", i)
		size := int64((i + 1) * 5)
		assert.False(t, file.IsDir(), "dir")
		assert.Equal(t, size, file.Size(), "size")
		assert.Equal(t, name, file.Name(), "name")
	}
}

func TestZipWriter_dirs(t *testing.T) {
	var buf bytes.Buffer
	zip := archive.NewZip(&buf)
	assert.NoError(t, zip.Open(), "open")

	for i := 0; i < 10; i++ {
		s := strings.Repeat("Hello", i+1)
		d := strings.Repeat("foo/", i)
		name := fmt.Sprintf("%sexample-%d.txt", d, i)

		a, err := zip.Add(archive.Info{
			Name: name,
			Size: int64(len(s)),
			Mode: 0700,
		}.FileInfo())

		assert.NoError(t, err, "add")

		_, err = io.WriteString(a, s)
		assert.NoError(t, err, "write")
	}

	assert.NoError(t, zip.Close(), "open")

	dir := unzip(t, &buf)

	s, err := tree(dir)
	assert.NoError(t, err, "tree")

	expected := `example-0.txt mode=-rwx------ size=5
foo mode=drwxr-xr-x
foo/example-1.txt mode=-rwx------ size=10
foo/foo mode=drwxr-xr-x
foo/foo/example-2.txt mode=-rwx------ size=15
foo/foo/foo mode=drwxr-xr-x
foo/foo/foo/example-3.txt mode=-rwx------ size=20
foo/foo/foo/foo mode=drwxr-xr-x
foo/foo/foo/foo/example-4.txt mode=-rwx------ size=25
foo/foo/foo/foo/foo mode=drwxr-xr-x
foo/foo/foo/foo/foo/example-5.txt mode=-rwx------ size=30
foo/foo/foo/foo/foo/foo mode=drwxr-xr-x
foo/foo/foo/foo/foo/foo/example-6.txt mode=-rwx------ size=35
foo/foo/foo/foo/foo/foo/foo mode=drwxr-xr-x
foo/foo/foo/foo/foo/foo/foo/example-7.txt mode=-rwx------ size=40
foo/foo/foo/foo/foo/foo/foo/foo mode=drwxr-xr-x
foo/foo/foo/foo/foo/foo/foo/foo/example-8.txt mode=-rwx------ size=45
foo/foo/foo/foo/foo/foo/foo/foo/foo mode=drwxr-xr-x
foo/foo/foo/foo/foo/foo/foo/foo/foo/example-9.txt mode=-rwx------ size=50
`

	assert.Equal(t, expected, s)
}

func TestZipWriter_transform(t *testing.T) {
	var buf bytes.Buffer
	zip := archive.NewZip(&buf)
	assert.NoError(t, zip.Open(), "open")

	// TODO: dirs as well?
	zip.WithTransform(archive.TransformFunc(func(r io.Reader, i os.FileInfo) (io.Reader, os.FileInfo) {
		info := archive.Info{
			Name:     i.Name(),
			Size:     i.Size(),
			Mode:     i.Mode() | 0777,
			Modified: i.ModTime(),
			Dir:      i.IsDir(),
		}

		return r, info.FileInfo()
	}))

	err := zip.AddDir(filepath.Join("testdata", "static"))
	assert.NoError(t, err, "add dir")
	assert.NoError(t, zip.Close(), "open")

	dir := unzip(t, &buf)

	s, err := tree(dir)
	assert.NoError(t, err, "tree")

	expected := `testdata mode=drwxr-xr-x
testdata/static mode=drwxr-xr-x
testdata/static/index.html mode=-rwxrwxrwx size=117
testdata/static/style.css mode=-rwxrwxrwx size=44
`

	assert.Equal(t, expected, s)
}

func BenchmarkZipWriter(b *testing.B) {
	max := 250 << 10 // 250k
	step := 25 << 10 // 25k

	for i := step; i <= max; i += step {
		size := i / (1 << 10)
		name := fmt.Sprintf("%dKiB", size)
		val := strings.Repeat("a", i)

		b.Run(name, func(b *testing.B) {
			zip := archive.New(archive.Zip, ioutil.Discard)
			assert.NoError(b, zip.Open(), "open")

			for i := 0; i < b.N; i++ {
				file, err := zip.Add(archive.Info{
					Name: "example-" + strconv.Itoa(i),
					Size: int64(len(val)),
				}.FileInfo())

				_, err = io.WriteString(file, val)
				assert.NoError(b, err, "write")
			}

			assert.NoError(b, zip.Close(), "open")
		})
	}
}
