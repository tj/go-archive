package archive

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/denormal/go-gitignore"
)

// Filter is the interface used to filter on files.
type Filter interface {
	// Match on the given file info, if the function
	// returns true then the file is omitted.
	Match(os.FileInfo) bool
}

// FilterFunc implements the Filter interface.
type FilterFunc func(os.FileInfo) bool

// Match implementation.
func (f FilterFunc) Match(i os.FileInfo) bool {
	return f(i)
}

// FilterDotfiles filters dotfiles.
var FilterDotfiles = FilterFunc(func(info os.FileInfo) bool {
	dir, file := filepath.Split(info.Name())
	return isDot(dir) || isDot(file)
})

// isDot returns true if there's a leading dot.
func isDot(s string) bool {
	return len(s) > 0 && s[0] == '.'
}

// FilterPatterns filters on the given reader.
func FilterPatterns(r io.Reader) (Filter, error) {
	filter := gitignore.New(r, ".", func(e gitignore.Error) bool {
		return true
	})

	return FilterFunc(func(info os.FileInfo) bool {
		if m := filter.Relative(info.Name(), info.IsDir()); m != nil {
			return m.Ignore()
		}
		return false
	}), nil
}

// FilterPatternFiles filters from the given files, ignoring
// any which do not exist, combining the patterns in order.
func FilterPatternFiles(files ...string) (Filter, error) {
	var r io.Reader = strings.NewReader("")

	for _, path := range files {
		b, err := ioutil.ReadFile(path)

		if os.IsNotExist(err) {
			continue
		}

		if err != nil {
			return nil, err
		}

		r = io.MultiReader(r,
			strings.NewReader(fmt.Sprintf("# %s\n", path)),
			bytes.NewReader(b),
			strings.NewReader("\n"))
	}

	return FilterPatterns(r)
}
