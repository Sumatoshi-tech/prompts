// Package promptkit provides the promptkit scaffold CLI and embedded templates.
package promptkit

import "embed"

// Templates holds all embedded template files for project scaffolding.
//
//go:embed all:templates
var Templates embed.FS
