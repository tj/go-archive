package archive

import (
	"strings"
	"testing"

	"github.com/tj/assert"
)

type filterCase struct {
	Info Info
	Ok   bool
}

type filterCases []filterCase

func (cases filterCases) Test(t *testing.T, f Filter) {
	for _, c := range cases {
		info := c.Info.FileInfo()
		included := c.Ok

		t.Run(info.Name(), func(t *testing.T) {
			includedResult := !f.Match(info)

			if included == includedResult {
				return
			}

			s := "be filtered"
			if included {
				s = "not be filtered"
			}

			t.Fatalf("expected %q to %s", info.Name(), s)
		})
	}
}

func file(name string, ok bool) filterCase {
	return filterCase{
		Info: Info{
			Name: name,
		},
		Ok: ok,
	}
}

func TestFilterDotfiles(t *testing.T) {
	cases := filterCases{
		file("foo", true),
		file("foo/bar/baz", true),
		file(".envrc", false),
		file("build/.something", false),
		file(".git", false),
		file(".git/hooks", false),
		file(".git/hooks/pre-commit", false),
	}

	cases.Test(t, FilterDotfiles)
}

func TestFilterPatterns(t *testing.T) {
	cases := filterCases{
		file("server", true),
		file("main.go", false),
		file("Readme.md", false),
		file(".git", false),
	}

	patterns := strings.NewReader(`
.git
*.md
*.go
`)

	f, err := FilterPatterns(patterns)
	assert.NoError(t, err, "filter")

	cases.Test(t, f)
}

func TestFilterPatterns_negate(t *testing.T) {
	cases := filterCases{
		file("server", true),
		file("main.go", false),
		file("Readme.md", false),
		file(".git", false),
	}

	patterns := strings.NewReader(`
*
!server
`)

	f, err := FilterPatterns(patterns)
	assert.NoError(t, err, "filter")

	cases.Test(t, f)
}

func TestFilterPattern_deeply_nested(t *testing.T) {
	cases := filterCases{
		file(".git", false),
		file("readme.md", false),
		file("server", true),

		file("node_modules", true),

		file("package.json", false),
		file("node_modules/foo/package.json", true),

		file("src/main.go", false),
		file("node_modules/bar/src", true),
		file("node_modules/bar/src/index.js", true),
	}

	patterns := strings.NewReader(`
.git
readme.md
src/**
package.json
!node_modules/**
!server
`)

	f, err := FilterPatterns(patterns)
	assert.NoError(t, err, "filter")

	cases.Test(t, f)
}

func TestFilterPatternFiles(t *testing.T) {
	cases := filterCases{
		file("server", true),
		file("static", true),
		file("static/index.html", true),
		file("node_modules/foo", false),
		file("node_modules/bar", false),
		file("client/build", false),
		// file("logs/foo.log", false),
		// file("logs/foo/bar.log", false),
		// file("logs/foo/baz.log", false),
		file("main.go", false),
		file("Readme.md", false),
		file("package.json", false),
		file(".git", false),
		file(".envrc", false),
		file(".foo", false),
		file(".bar", true),
	}

	f, err := FilterPatternFiles("testdata/.gitignore", "testdata/nope", "testdata/.npmignore", "testdata/.upignore")
	assert.NoError(t, err, "filter")

	cases.Test(t, f)
}

func BenchmarkFilter(b *testing.B) {
	b.Run("FilterDotfiles", func(b *testing.B) {
		f := FilterDotfiles

		info := Info{
			Name: "something",
		}.FileInfo()

		for i := 0; i < b.N; i++ {
			f.Match(info)
		}
	})
}
