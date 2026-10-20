// Package web contains the widget's build-free browser assets.
package web

import "embed"

// Assets contains only public browser files, never tests.
//
//go:embed assets
var Assets embed.FS
