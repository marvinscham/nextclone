package locales

import "embed"

// FS contains the translation files used by the application.
//
//go:embed *.json
var FS embed.FS
