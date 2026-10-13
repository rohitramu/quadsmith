//go:build test

package main

import _ "embed"

//go:embed config/test.textproto
var embeddedConfigRaw []byte
