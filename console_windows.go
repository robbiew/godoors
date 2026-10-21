//go:build windows

package godoors

import (
	"sync"

	"golang.org/x/sys/windows"
)

var consoleOnce sync.Once

// prepareConsole turns on ANSI escape processing for a local Windows
// console (older conhost windows print escape codes literally without it).
// The process's real stdout handle is used, so it still applies when a
// door has redirected os.Stdout through a filter. No-op for pipes and
// sockets (the BBS case).
func prepareConsole() {
	consoleOnce.Do(func() {
		h, err := windows.GetStdHandle(windows.STD_OUTPUT_HANDLE)
		if err != nil {
			return
		}
		var mode uint32
		if windows.GetConsoleMode(h, &mode) != nil {
			return
		}
		_ = windows.SetConsoleMode(h, mode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING)
	})
}
