// Package assets contains documentation shipped inside the standalone binary.
package assets

import _ "embed"

//go:embed skills/SKILL.md
var Skill string
