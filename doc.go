// Package godoors is a helper library for building Linux-based BBS door
// programs (games and utilities) that talk to a caller over STDIN/STDOUT
// through a terminal such as SyncTerm, MagiTerm, NetRunner or IGTerm.
//
// It covers the common door chores: reading a door32.sys drop file, detecting
// the terminal size, drawing SAUCE-stripped ANSI art, positioning the cursor,
// applying pipe/ANSI colors, and simple prompts (pause, yes/no, idle timeout).
//
// The source is organized by concern:
//
//	ansi.go     - escape-sequence, color, font and symbol constants
//	cursor.go   - cursor movement and screen control
//	terminal.go - terminal size detection
//	dropfile.go - door32.sys parsing
//	user.go     - the User type, Initialize, and centered/modal drawing
//	art.go      - ANSI art and text output
//	sauce.go    - SAUCE metadata trimming
//	color.go    - pipe-code ("|00") color rendering
//	prompt.go   - key prompts and the idle timer
package godoors
