//go:build windows || !cgo

package postgresql

import (
	nodes "github.com/wasilibs/go-pgquery"
)

var Parse = nodes.Parse
var ParseScan = nodes.Scan
var Fingerprint = nodes.Fingerprint
var Deparse = nodes.Deparse
