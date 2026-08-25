// Package ckadojo carries the training content embedded into the dojo binary.
//
// The engine reads content through an fs.FS so a colleague can run a single
// binary, while development works against the repository via DOJO_CONTENT.
package ckadojo

import "embed"

//go:embed all:environments all:curriculum
var Content embed.FS
