package godoors

import (
	"fmt"
	"os"
	"time"

	"github.com/eiannone/keyboard"
)

// Idle is the number of seconds a user may sit at a Pause/Continue prompt
// before IdleAction fires. A value of 0 (or negative) disables the timeout.
var Idle int

// IdleAction runs when the user idles past Idle seconds at a prompt. It
// defaults to the usual door behavior — print a notice and exit, so the BBS
// can reclaim the node — but callers may override it to run their own cleanup,
// logging, or a non-terminating handler.
var IdleAction = func() {
	fmt.Println("\r\nYou've been idle for too long... exiting!")
	time.Sleep(1 * time.Second)
	os.Exit(0)
}

// startIdleTimer arms the idle timer when Idle is configured (> 0), returning a
// stop function that is always safe to defer. When Idle is 0 or negative the
// timer is disabled, so a caller who never sets Idle is not booted the instant
// they hit a prompt.
func startIdleTimer() func() {
	if Idle <= 0 {
		return func() {}
	}
	t := NewTimer(Idle, func() { IdleAction() })
	return func() { t.Stop() }
}

// Continue reads a single key and reports whether the user chose to proceed.
// Y/y/Enter mean yes; N/n/Esc mean no; any other key re-prompts. If the
// keyboard can't be read it returns false rather than crashing the door.
func Continue() bool {
	defer startIdleTimer()()

	for {
		char, key, err := keyboard.GetKey()
		if err != nil {
			return false
		}
		switch {
		case string(char) == "Y" || string(char) == "y" || key == keyboard.KeyEnter:
			return true
		case string(char) == "N" || string(char) == "n" || key == keyboard.KeyEsc:
			return false
		}
	}
}

// NewTimer runs action after the given number of seconds and returns the
// underlying timer so the caller can cancel it with Stop. It uses
// time.AfterFunc, so stopping the timer before it fires leaves no goroutine
// blocked waiting on the channel.
func NewTimer(seconds int, action func()) *time.Timer {
	return time.AfterFunc(time.Duration(seconds)*time.Second, action)
}

// Pause prints a prompt and waits for a single key press. A keyboard read
// error is ignored so a failing terminal can't crash the door.
func Pause() {
	defer startIdleTimer()()

	fmt.Fprint(os.Stdout, "\r\nPrEsS a KeY")
	_, _, _ = keyboard.GetKey()
}
