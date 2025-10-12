package vcs

import (
	// "fmt"
	"runtime/debug"
)

func Version() string {
	bi, ok := debug.ReadBuildInfo()
	if ok {
		return bi.Main.Version
	}

	return ""
}