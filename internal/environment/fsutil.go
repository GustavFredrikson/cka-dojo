package environment

import (
	"io/fs"

	"github.com/gustavfredrikson/cka-dojo/internal/content"
)

func fsReadDir(src *content.Source, dir string) ([]fs.DirEntry, error) {
	return fs.ReadDir(src.FS, dir)
}
