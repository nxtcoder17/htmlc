package examples

import (
	"embed"
)

//go:embed htmlc.yml components pages
var ExamplesFS embed.FS
