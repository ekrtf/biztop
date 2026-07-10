// Package gui embeds the frontend assets into the binary.
package gui

import "embed"

//go:embed index.html style.css js
var FS embed.FS
