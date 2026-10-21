package godoors

import "strconv"

// Screen coordinates in godoors are 1-based, as in ANSI: (1, 1) is the
// top-left cell, x is the column and y is the row. Values below 1 are
// clamped to 1, so (0, 0) also means the top-left cell. Callers that keep
// 0-based coordinates should add 1 when calling in.

// clampCoord pins a coordinate to the 1-based minimum.
func clampCoord(n int) int {
	if n < 1 {
		return 1
	}
	return n
}

// cursorTo returns the escape sequence that moves the cursor to column x,
// row y (1-based, clamped).
func cursorTo(x, y int) string {
	return Esc + strconv.Itoa(clampCoord(y)) + ";" + strconv.Itoa(clampCoord(x)) + "f"
}
