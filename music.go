package godoors

import (
	"fmt"
	"os"
	"strings"
)

// ANSI music, as played by SyncTERM and other CTerm-compatible terminals
// (check User.CTerm or IsCTerm first; other terminals may print it).
//
// A tune is written in the classic BASIC PLAY / MML notation, for example
// "T120 L8 O4 C D E F G4 P8 >C2":
//
//	A-G       note; follow with + or # for sharp, - for flat, a number for
//	          its length (4 = quarter) and . for dotted
//	O n, < >  set the octave (0-6), or step down / up one
//	L n       default note length
//	T n       tempo in quarter notes per minute
//	P n, N n  pause; N n plays note number n (1-84, 0 = rest)
//	MF / MB   play in the foreground (wait) / background (don't wait)
//	MN ML MS  normal, legato, staccato
//
// The sequence sent is CSI | <tune> SO. SyncTERM always accepts the CSI |
// form, unlike the older CSI M / CSI N forms, which clash with "delete
// line" and depend on the terminal's settings.

// MusicSequence wraps a tune in the ANSI music introducer and terminator.
// Control characters (which would end or corrupt the sequence early) are
// dropped from the tune.
func MusicSequence(tune string) string {
	clean := strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, tune)
	return "\x1b[|" + clean + "\x0e"
}

// PlayMusic sends a tune to the terminal as ANSI music.
func PlayMusic(tune string) {
	fmt.Fprint(os.Stdout, MusicSequence(tune))
}
