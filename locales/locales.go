// Package locales embeds all translation string files.
// Add a new language by dropping a <lang>.json file here (e.g. en.json).
package locales

import "embed"

//go:embed *.json
var Files embed.FS
