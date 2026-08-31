// Package assets embeds game art so desktop and WASM builds ship identical
// content with no runtime file loading.
package assets

import "embed"

// TileFS holds plot and crop art. The scaffold renders with vector
// primitives and ships no art yet; the `all:` prefix keeps the embed valid
// while the directory holds only a .gitkeep.
//
//go:embed all:images
var TileFS embed.FS
