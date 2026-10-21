package godoors

import (
	"fmt"
	"os"
	"strings"
)

// ANSI music, as played by SyncTERM and other CTerm-compatible terminals
// (check User.CTerm or IsCTerm first; other terminals may print it).
//
// A tune uses the CTerm music grammar (a dialect of BASIC PLAY), for
// example "T120 L8 O4 C D E F G4 P8 >C2":
//
//	A-G       note; + or # sharpens and - flattens (one semitone each),
//	          then an optional length (4 = quarter) and . for dotted
//	P         pause, with the same length rules as a note
//	O n, < >  set the octave (0-6, default 4), or step down / up one
//	L n       default length for notes without one (1-64, default 4)
//	T n       tempo in quarter notes per minute (32-255, default 120)
//	N n       play note number n (0-71; 0 is C in octave 0)
//	MF / MB   play in the foreground (wait) / background (don't wait)
//	MN ML MS  normal, legato, staccato
//
// Only the characters in musicChars are allowed; CTerm ignores a whole
// tune containing anything else, so MusicSequence drops other characters.
//
// The sequence sent is CSI | <tune> SO SI. CTerm always accepts the CSI |
// introducer (the older CSI M and CSI N forms depend on terminal settings,
// and CSI M clashes with "delete line"). SO (0x0E) ends the tune; SI
// (0x0F) follows it, as CTerm requires, so a terminal that doesn't play
// music isn't left shifted into its alternate character set.

// musicChars is CTerm's list of characters allowed in a music string.
const musicChars = "aAbBcCdDeEfFgGlLmMnNoOpPsStT0123456789.-+#<> "

// MusicSequence wraps a tune in the ANSI music introducer and terminator,
// dropping any character CTerm doesn't allow in a tune.
func MusicSequence(tune string) string {
	clean := strings.Map(func(r rune) rune {
		if !strings.ContainsRune(musicChars, r) {
			return -1
		}
		return r
	}, tune)
	return "\x1b[|" + clean + "\x0e\x0f"
}

// PlayMusic sends a tune to the terminal as ANSI music.
func PlayMusic(tune string) {
	fmt.Fprint(os.Stdout, MusicSequence(tune))
}
