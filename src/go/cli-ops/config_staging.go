//go:build staging

package main

import _ "embed"

//go:embed config/staging.textproto
var embeddedConfigRaw []byte
