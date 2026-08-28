//go:build embedui

package dashboard

import "embed"

//go:embed all:web/build
var embeddedWeb embed.FS

const embeddedWebRoot = "web/build"
