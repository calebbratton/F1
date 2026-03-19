// Package web embeds the dashboard static files into the binary.
package web

import "embed"

//go:embed static
var Static embed.FS
