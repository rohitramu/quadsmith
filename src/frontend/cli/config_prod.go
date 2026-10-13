//go:build prod

package main

import _ "embed"

//go:embed config/prod.textproto
var embeddedConfigRaw []byte
