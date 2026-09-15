// Package gonext holds the embedded templates/ tree used by the scaffolding
// CLI. It lives at the repository root because go:embed cannot ascend
// directories.
package gonext

import "embed"

//go:embed all:templates
var Templates embed.FS
