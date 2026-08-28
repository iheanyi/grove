//go:build !embedui

package dashboard

import "embed"

//go:embed all:web/stub
var embeddedWeb embed.FS

const embeddedWebRoot = "web/stub"
