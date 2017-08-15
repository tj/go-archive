package archive_test

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/tj/assert"
	archive "github.com/tj/go-archive"
)

func TestFilterDotfiles(t *testing.T) {
	os.Chdir("testdata/node")
	defer os.Chdir("../..")

	var buf bytes.Buffer
	a := archive.NewZip(&buf).WithFilter(archive.FilterDotfiles)
	assert.NoError(t, a.Open(), "open")
	assert.NoError(t, a.AddDir("."), "add dir")
	assert.NoError(t, a.Close(), "close")

	dir := unzip(t, &buf)
	s, err := tree(dir)
	assert.NoError(t, err, "tree")

	expected := `Readme.md mode=-rw-r--r-- size=0
app.js mode=-rw-r--r-- size=0
package.json mode=-rw-r--r-- size=0
up.json mode=-rw-r--r-- size=0
`

	assert.Equal(t, expected, s)
}

func TestFilterPatterns_dir(t *testing.T) {
	os.Chdir("testdata/node")
	defer os.Chdir("../..")

	patterns := strings.NewReader(`
*.md
.something
`)

	f, err := archive.FilterPatterns(patterns)
	assert.NoError(t, err, "filter")

	var buf bytes.Buffer
	a := archive.NewZip(&buf).WithFilter(f)
	assert.NoError(t, a.Open(), "open")
	assert.NoError(t, a.AddDir("."), "add dir")
	assert.NoError(t, a.Close(), "close")

	dir := unzip(t, &buf)
	s, err := tree(dir)
	assert.NoError(t, err, "tree")

	expected := `app.js mode=-rw-r--r-- size=0
package.json mode=-rw-r--r-- size=0
up.json mode=-rw-r--r-- size=0
`

	assert.Equal(t, expected, s)
}

func TestFilterPatterns_negated(t *testing.T) {
	os.Chdir("testdata/node")
	defer os.Chdir("../..")

	patterns := strings.NewReader(`
*.md
*.json
!up.json
.something
!.something
`)

	f, err := archive.FilterPatterns(patterns)
	assert.NoError(t, err, "filter")

	var buf bytes.Buffer
	a := archive.NewZip(&buf).WithFilter(f)
	assert.NoError(t, a.Open(), "open")
	assert.NoError(t, a.AddDir("."), "add dir")
	assert.NoError(t, a.Close(), "close")

	dir := unzip(t, &buf)
	s, err := tree(dir)
	assert.NoError(t, err, "tree")

	expected := `.something mode=drwxr-xr-x
.something/bar mode=drwxr-xr-x
.something/bar/baz mode=drwxr-xr-x
.something/bar/baz/something mode=-rw-r--r-- size=0
.something/foo mode=-rw-r--r-- size=0
app.js mode=-rw-r--r-- size=0
up.json mode=-rw-r--r-- size=0
`

	assert.Equal(t, expected, s)
}

func TestFilterPatterns_nested(t *testing.T) {
	os.Chdir("testdata/node")
	defer os.Chdir("../..")

	patterns := strings.NewReader(`
*.md
*.json
!up.json
.something/bar/**
`)

	f, err := archive.FilterPatterns(patterns)
	assert.NoError(t, err, "filter")

	var buf bytes.Buffer
	a := archive.NewZip(&buf).WithFilter(f)
	assert.NoError(t, a.Open(), "open")
	assert.NoError(t, a.AddDir("."), "add dir")
	assert.NoError(t, a.Close(), "close")

	dir := unzip(t, &buf)
	s, err := tree(dir)
	assert.NoError(t, err, "tree")

	expected := `.something mode=drwxr-xr-x
.something/foo mode=-rw-r--r-- size=0
app.js mode=-rw-r--r-- size=0
up.json mode=-rw-r--r-- size=0
`

	assert.Equal(t, expected, s)
}

func TestFilterPatterns_allNegated(t *testing.T) {
	os.Chdir("testdata/node")
	defer os.Chdir("../..")

	patterns := strings.NewReader(`
*
!up.json
!.something/**
`)

	f, err := archive.FilterPatterns(patterns)
	assert.NoError(t, err, "filter")

	var buf bytes.Buffer
	a := archive.NewZip(&buf).WithFilter(f)
	assert.NoError(t, a.Open(), "open")
	assert.NoError(t, a.AddDir("."), "add dir")
	assert.NoError(t, a.Close(), "close")

	dir := unzip(t, &buf)
	s, err := tree(dir)
	assert.NoError(t, err, "tree")

	expected := `.something mode=drwxr-xr-x
.something/bar mode=drwxr-xr-x
.something/bar/baz mode=drwxr-xr-x
.something/bar/baz/something mode=-rw-r--r-- size=0
.something/foo mode=-rw-r--r-- size=0
up.json mode=-rw-r--r-- size=0
`

	assert.Equal(t, expected, s)
}
