//go:build tools
// +build tools

package tools

// Required for go:generate to work
import (
	_ "github.com/go-bindata/go-bindata"
)
