// Package content resolves where curriculum and environment definitions are
// read from: an explicit directory when developing, the embedded copy
// otherwise.
package content

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	ckadojo "github.com/gustavfredrikson/cka-dojo"
)

// Source is a content root containing environments/ and curriculum/.
type Source struct {
	FS fs.FS
	// Origin describes where the content came from, for dojo doctor.
	Origin string
}

// Resolve picks a content root. Precedence: explicit dir, DOJO_CONTENT, the
// configured dir, then the content embedded in the binary.
func Resolve(explicit, configured string) (*Source, error) {
	for _, cand := range []string{explicit, os.Getenv("DOJO_CONTENT"), configured} {
		if cand == "" {
			continue
		}
		abs, err := filepath.Abs(cand)
		if err != nil {
			return nil, err
		}
		if _, err := os.Stat(filepath.Join(abs, "curriculum")); err != nil {
			return nil, fmt.Errorf("content dir %s has no curriculum/ directory", abs)
		}
		return &Source{FS: os.DirFS(abs), Origin: abs}, nil
	}
	return &Source{FS: ckadojo.Content, Origin: "embedded"}, nil
}

// Read returns the bytes of a content file.
func (s *Source) Read(name string) ([]byte, error) { return fs.ReadFile(s.FS, name) }

// Exists reports whether a content path is present.
func (s *Source) Exists(name string) bool {
	_, err := fs.Stat(s.FS, name)
	return err == nil
}
