//go:build !windows

package godoors

// prepareConsole is a no-op outside Windows: Unix terminals already
// process ANSI escapes.
func prepareConsole() {}
