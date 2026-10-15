//go:build sandbox || (!test && !staging && !prod)

package main

import _ "embed"

//go:embed config/sandbox.textproto
var embeddedConfigRaw []byte
