package godoors

import "fmt"

// MoveCursor moves the cursor to column x, row y. Coordinates are 1-based
// ((1, 1) is the top-left cell); values below 1 are clamped to 1.
func MoveCursor(x int, y int) {
	fmt.Fprint(Output(), cursorTo(x, y))
}

// Erase the screen and home the cursor.
func ClearScreen() {
	fmt.Fprintln(Output(), EraseScreen)
	MoveCursor(1, 1)
}

// Move the cursor n cells to up.
func CursorUp(n int) {
	fmt.Fprintf(Output(), Esc+"%dA", n)
}

// Move the cursor n cells to down.
func CursorDown(n int) {
	fmt.Fprintf(Output(), Esc+"%dB", n)
}

// Move the cursor n cells to right.
func CursorForward(n int) {
	fmt.Fprintf(Output(), Esc+"%dC", n)
}

// Move the cursor n cells to left.
func CursorBack(n int) {
	fmt.Fprintf(Output(), Esc+"%dD", n)
}

// Move cursor to beginning of the line n lines down.
func CursorNextLine(n int) {
	fmt.Fprintf(Output(), Esc+"%dE", n)
}

// Move cursor to beginning of the line n lines up.
func CursorPreviousLine(n int) {
	fmt.Fprintf(Output(), Esc+"%dF", n)
}

// Move cursor horizontally to column x (1-based, clamped to 1).
func CursorHorizontalAbsolute(x int) {
	fmt.Fprintf(Output(), Esc+"%dG", clampCoord(x))
}

// Show the cursor.
func CursorShow() {
	fmt.Fprint(Output(), Esc+"?25h")
}

// Hide the cursor.
func CursorHide() {
	fmt.Fprint(Output(), Esc+"?25l")
}

// Save the screen.
func SaveScreen() {
	fmt.Fprint(Output(), Esc+"?47h")
}

// Restore the saved screen.
func RestoreScreen() {
	fmt.Fprint(Output(), Esc+"?47l")
}
