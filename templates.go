// Package gonext holds the embedded templates/ tree used by the
// scaffolding CLI. It must live at the repository root because
// go:embed patterns cannot ascend directories, and templates/ is a
// root-level sibling of cmd/ and internal/.
package gonext

import "embed"

//go:embed all:templates
var Templates embed.FS
