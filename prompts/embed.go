// Package prompts contains the user-editable prompt templates embedded in the
// CommitGen executable at build time.
package prompts

import _ "embed"

// Title is the prompt used to generate a Conventional Commit title.
//
//go:embed title.txt
var Title string

// Description is the prompt used to generate a commit description.
//
//go:embed description.txt
var Description string
