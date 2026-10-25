# :computer: GoDoors

Helpful library for creating door applications (like games and utilities) for BBSs on Linux and macOS through STDIN/STDOUT, and on current Windows through local standard I/O or inherited Door32 BBS sockets — when connected over a terminal program like [SyncTerm](https://syncterm.bbsdev.net/), [MagiTerm](https://gitlab.com/magickabbs/MagiTerm), [NetRunner](http://mysticbbs.com/downloads.html) or [IGTerm](https://www.phenomprod.com/).

----

## INSTALL
```go
go get github.com/robbiew/godoors
```

## USAGE
```go
import (
    gd "github.com/robbiew/godoors"
)
```


## DROP FILES

```go
gd.DropFileData(path string) (alias string, timeLeft, emulation, node int, err error)
```

> :point_up: Pass the path of the folder **containing** a [door32.sys](https://raw.githubusercontent.com/NuSkooler/ansi-bbs/master/docs/dropfile_formats/door32_sys.txt) drop file (a trailing slash is optional). It returns HANDLE/ALIAS, TIME LEFT (in minutes), EMULATION TYPE (0 = ASCII, 1 = ANSI), NODE NUMBER, and an `error`. The filename is matched case-insensitively (`door32.sys`, `DOOR32.SYS`, `Door32.Sys`). Experimental DOORDROP.INI draft 0.8 support is also available via an explicit file path (see below).

```go
alias, timeLeft, emulation, node, err := gd.DropFileData("./")
if err != nil {
    log.Fatal(err)
}
```

## EXPERIMENTAL DOORDROP.INI

Support for [DOORDROP.INI draft 0.8](https://github.com/SynchronetBBS/sbbs/blob/master/docs/doordrop_ini.md)
is **experimental until the specification reaches v1.0**. Draft fields and this
API may change. This implements a subset for GoDoors' existing byte-oriented I/O.

Pass the exact `.ini` filename to `ReadDropFile`, `DropFileData`, `OpenSession`,
or `Initialize`. Pass an empty string to use `DOORDROP_INI`. An explicit path
wins over the environment; existing directory arguments still select Door32.
GoDoors does not search a directory for DOORDROP.INI or read command-line
arguments for you.

```go
info, err := gd.OpenSession(os.Getenv("DOORDROP_INI"))
if err != nil {
    log.Fatal(err)
}
defer gd.CloseSession()
if info.INI != nil {
    userKey := info.INI.Values["USER_KEY"]
    _ = userKey // use a stable identity for saved progress
}
```

`COMM_TYPE=local` and `stdio` are supported on all platforms. Plain `socket`
uses the inherited socket on Windows, with invalid handles returned as errors.
POSIX socket descriptors, raw `telnet`, serial, FOSSIL and UART are currently
unsupported and fail explicitly when opening a session. Reading socket metadata
does not open or validate the socket. Remote connections must specify
`COMM_CHARSET=CP437`; UTF-8 file text (`FILE_UTF8=1`) is currently rejected.
Text remains raw CP437, including alias bytes.

The existing alias, node and emulation fields are populated (`ansi` and `avatar`
map to ANSI; unknown terminal types map to ASCII). `TIME_LEFT` seconds round up
to legacy `TimeLeft` minutes. `info.INI.TimeLeftSeconds` preserves exact seconds:
`nil` means unlimited, and a pointer to zero means expired. Applications must
enforce this limit themselves, as with Door32 time limits. `INI.Values` exposes
first-occurrence values, including vendor keys. Unknown keys and lines without
`=` are ignored; inline `;`, `#`, quotes and `=` remain literal. CRLF, LF and
DOS Ctrl-Z end-of-file are accepted.

Only fields used by GoDoors are validated. Other values in `INI.Values` are raw:
applications must apply the draft's defaults and validate any additional keys
they use. Terminal geometry and CTerm detection still use the existing probes;
preferences such as `PREF_SOUND`, pause, and idle limits are exposed for the
application to honor and are not automatically applied by GoDoors helpers.

## INITIALIZE

```go
gd.Initialize(path string) (gd.User, error)
```

> :point_up: Convenience wrapper: reads the drop file **and** probes the terminal in one call, returning a populated `User` (Alias, TimeLeft, Emulation, NodeNum, H, W, ModalH, ModalW, CTerm). `CTerm` is true when the caller's terminal identifies as SyncTERM/CTerm (see [ANSI MUSIC](#ansi-music)). Returns an `error` if the drop file can't be read or parsed, or the Windows connection can't be opened. After a successful call, use `defer gd.CloseSession()` to release the connection and restore local input mode.

***
 
## OPTIONAL INITIALIZATION AND EXTENDED KEYS

`gd.OpenSession(path)` selects the dropfile connection and returns a `DropFile`
without terminal probes or input reads. Pair it with `defer gd.CloseSession()`.
Call `gd.GetTermSize()` separately when detection is wanted. `Initialize` keeps
its existing connection-plus-detection behavior.

`gd.ReadKey()` shares the existing input buffer and returns the same rune/key/error
triple as `GetKey`, adding PageUp/PageDown, Home/End, Insert/Delete, and F1–F12
key events. `GetKey` retains its original arrows-only handling of escape sequences.
Use `gd.ReadKeyTimeout(duration)` for a bounded wait; timeout returns
`gd.ErrInputTimeout` and preserves later input for the next read.

## WINDOWS BBS DOORS AND SESSION I/O

`Initialize` selects the connection before probing the caller's terminal. On
Windows, Door32 communication type `2` uses the inherited socket handle from
line 2 of `door32.sys`. A zero or non-socket handle keeps the existing
stdin/stdout transport for hosts that redirect standard I/O. Other socket errors
are returned. Type `0` uses local stdin/stdout; Windows serial/type `1`
is unsupported and returns an error. On Linux and macOS, standard I/O remains
the transport, including when the drop file describes a telnet connection.

```go
func runDoor(dropDirectory string) error {
    u, err := gd.Initialize(dropDirectory)
    if err != nil {
        return err
    }
    defer gd.CloseSession()
    if err := gd.OpenInput(); err != nil {
        return err
    }
    if _, err := fmt.Fprintf(gd.Output(), "Hello, %s!\r\n", u.Alias); err != nil {
        return err
    }
    gd.Pause()
    return nil
}
```

All GoDoors drawing, prompt, music, and keyboard functions use the active
connection. Use `gd.Output()` for your application's own text too: `fmt.Print`
still goes to the process's local stdout. Raw CP437 bytes are passed through
unchanged. No stdout replacement, TCP reconnection, or telnet negotiation is
performed; the BBS must pass a usable connected socket and hand off the stream.

`gd.ReadDropFile(directory)` additionally exposes `CommType` and `Handle` without
opening a connection. The handle is a process-local inherited handle, not a port
number. The existing `DropFileData` metadata-only signature is unchanged.

There is one active session per process. Initialize it before reading keys or
starting other I/O, and close it after those operations finish. Call
`gd.CloseSession()` before another socket initialization or before `os.Exit`
(which skips deferred cleanup). `CloseInput` remains available for restoring local
raw mode; `CloseSession` also releases the child's socket handle. Neither shuts
down the parent's BBS connection. Retain no old `Output()` writer across sessions.

The socket adapter uses a private overlapped Winsock descriptor and event waits.
It preserves the parent's blocking mode and event registrations, and cancellation
applies only to the door's pending operations. Failed socket closes retain the
connection so `CloseSession` can be retried. Terminal probes retain their bounded
wait; socket writes wait for progress or session closure. Existing drawing helpers
retain their void signatures; use the writer directly to handle output errors.

Initialize a socket session before reading or probing standard input. If a local
stdin pump has already read input, socket initialization returns an error without
replacing the pump or discarding its buffered keys. `OpenInput` alone can precede
initialization because it does not start a read.

Windows 10 / Server 2016 or later is required by the Go 1.21 baseline. Windows 7
is out of scope. Windows socket tests include a real inherited-handle subprocess,
large binary output, disconnects, and cancellation; they require native Windows
execution. CI runs the full suite on Linux and macOS, Windows amd64 with race
detection, and Windows 386. The Windows suites include checks that the parent can
reuse the connection after the child exits. Live acceptance under the target BBS
is still required; cross-compilation alone does not establish BBS compatibility.

***
## GET TERMINAL HEIGHT AND WIDTH
```go
gd.GetTermSize() (int, int)
```

> :point_up: Tries to detect the user's terminal size. Returns HEIGHT and WIDTH. A local terminal is asked directly; a remote caller's terminal is probed with a cursor-position report. If the terminal doesn't answer within 1.5 seconds, or can't be detected, it defaults to 25 and 80. Keys the caller types while the size is being probed are kept for the next `GetKey`.

***
## ANSI MUSIC
```go
gd.IsCTerm() bool
gd.PlayMusic(tune string)
gd.MusicSequence(tune string) string
```

> :point_up: SyncTERM (and other CTerm-compatible terminals) play ANSI music. `IsCTerm` asks the terminal what it is (the `ESC[c` device-attributes query; CTerm answers with its own signature) and caches the answer; `Initialize` fills in `User.CTerm` the same way. Don't rely on the `TERM` environment variable for this — on a BBS it describes the server, not the caller.

> `PlayMusic` sends a tune in the CTerm music grammar (a dialect of BASIC `PLAY`) as `CSI | <tune> SO SI`. CTerm always accepts the `CSI |` introducer, and the `SI` after the terminator keeps terminals that don't play music from staying shifted into their alternate character set. `MusicSequence` returns that string instead of printing it. CTerm ignores a whole tune containing any character outside `aAbBcCdDeEfFgGlLmMnNoOpPsStT0123456789.-+#<>` and space, so other characters are dropped.

```go
if u.CTerm {
	gd.PlayMusic("MB T140 L8 O3 G4 >G4 E-4 D4 <B-4 G2")
}
```

| Command | Meaning |
|---|---|
| `A`–`G` | Note. `+`/`#` sharpen and `-` flattens, one semitone each; then an optional length (`4` = quarter) and `.` for dotted |
| `P` | Pause, with the same length rules as a note |
| `O n`, `<`, `>` | Set octave (0–6, default 4), or step down/up one |
| `L n` | Default length for notes without one (1–64, default 4) |
| `T n` | Tempo in quarter notes per minute (32–255, default 120) |
| `N n` | Play note number n (0–71; `N0` is C in octave 0; out-of-range values are silent) |
| `MF` / `MB` | Play in the foreground (wait) / background (keep going) |
| `MN` / `ML` / `MS` | Normal / legato / staccato |

***
## DISPLAY ANSI ART
```go
gd.PrintAnsi(art string, delay int, height int)
```

> :point_up: Pass the **contents** of an ANSI art file (e.g. `b, _ := os.ReadFile(path); gd.PrintAnsi(string(b), 40, u.H)`). It strips the SAUCE metadata trailer, then prints line by line up to `height` lines, with an optional `delay` in milliseconds (e.g. 40) to simulate slower speeds. `PrintAnsiLoc`, `AbsCenterArt` and `Modal` likewise take art contents, not a path.

```go

var (
	gd.Heart        
	gd.ArrowUpDown  
	gd.ArrowUp      
	gd.ArrowDown   
	gd.ArrowDownFat 
	gd.ArrowRight  
	gd.ArrowLeft    
	gd.Block       
)
```
> :point_up: Variables for printing individual CP437 symbols on the fly, e.g. `fmt.Fprintln(gd.Output(), gd.Heart)`. (TO-DO: add more!)

***
## DISPLAY SOMETHING AT X,Y COORDINATES
```go
gd.PrintAnsiLoc(file string, x int, y int)
gd.PrintStringLoc(text string, x int, y int)
```

> :point_up: Same as above, only it'll print the art to the screen at X,Y coordinates, incrementing the Y position after every line. Handy of you need to update the screen with art in a particular location without clearing and re-writing everything.

> :warning: **Coordinates are 1-based**, as in ANSI: `(1, 1)` is the top-left cell, `x` is the column and `y` is the row. This applies to `PrintAnsiLoc`, `PrintStringLoc`, `MoveCursor` and `CursorHorizontalAbsolute`. Values below 1 are clamped to 1. If your code counts from 0, add 1 when calling in; otherwise row 0 and row 1 land on the same line.

***
## PAUSE
```go
gd.Pause()
```

> :point_up: Hit any key 

***
## CONTINUE Y/N PROMPT
```go
gd.Continue() bool
```

> :point_up: Reads one key: `Y`/`y`/Enter return `true`, `N`/`n`/Esc return `false`, and any other key re-prompts. Respects the idle timeout (`gd.Idle`).

***
## POP UP STYLE MODAL
```go
u, err := gd.Initialize("./")
if err != nil {
	// handle error
} else {
	defer gd.CloseSession()
	u.Modal(art, text)
}
```

> :point_up: Displays background ANSI art (contents, not a path) centered on screen with `text` and a "Continue? Y/n" prompt. It's a method on the `User` returned by `gd.Initialize`, so it's sized to that session's terminal. Sizing is measured from the art and text themselves — no length argument needed.

***

## CENTER SOMETHING (text, art, etc.)
```go
u, err := gd.Initialize("./")
if err != nil {
	// handle error
} else {
	defer gd.CloseSession()
	u.AbsCenterText(s, c) // method on User; appends "Continue? Y/n" and blocks
	u.AbsCenterArt(art)   // method on User
}
gd.CenterText(s, w) // package function
```
> :point_up: "absolute center" being both vertically and horizontally centered based on the terminal height and width. `AbsCenterText`/`AbsCenterArt` are methods on the `User` from `gd.Initialize`, so they use that session's dimensions, and sizing is measured from the text/art itself (no length argument). `c` is a background color constant. Note `AbsCenterText` appends a "Continue? Y/n" prompt and blocks until the user answers. `AbsCenterArt` takes art contents, not a path.

Centering ignores ANSI SGR color/style codes when measuring width. Other ANSI
cursor controls, tabs, and Unicode wide/combining characters are not simulated;
use line-oriented art with one character per cell for automatic centering.
Art lines are not limited to the default 64 KiB scanner buffer.

Keyboard input is byte-oriented (including raw CP437 bytes). CR, LF, and CRLF
each represent one Enter key. An incomplete ANSI key sequence times out after
30 milliseconds and returns Esc; complete CSI/SS3 arrow sequences remain supported.

***

## Cursor related

```go
// Move the cursor n cells to up.
gd.CursorUp(n int) 

// Move the cursor n cells to down.
gd.CursorDown(n int) 

// Move the cursor n cells to right.
gd.CursorForward(n int) 

// Move the cursor n cells to left.
gd.CursorBack(n int) 

// Move cursor to beginning of the line n lines down.
gd.CursorNextLine(n int) 

// Move cursor to beginning of the line n lines up.
gd.CursorPreviousLine(n int) 

// Move cursor horizontally to x.
gd.CursorHorizontalAbsolute(x int) 

// Show the cursor.
gd.CursorShow() 

// Hide the cursor.
gd.CursorHide()
```
***
## Color
```go
// Text colors supported by BBS term programs
// usage: fmt.Fprint(gd.Output(), gd.Yellow)
gd.Black         
gd.Red          
gd.Green         
gd.Yellow     
gd.Blue        
gd.Magenta      
gd.Cyan         
gd.White         
gd.BlackHi   
gd.RedHi      
gd.GreenHi     
gd.YellowHi    
gd.BlueHi      
gd.MagentaHi   
gd.CyanHi     
gd.WhiteHi     

// Background colors
gd.BgBlack        
gd.BgRed          
gd.BgGreen        
gd.BgYellow       
gd.BgBlue          
gd.BgMagenta       
gd.BgCyan          
gd.BgWhite         
gd.BgBlackHi     
gd.BgRedHi       
gd.BgGreenHi     
gd.BgYellowHi   
gd.BgBlueHi     
gd.BgMagentaHi   
gd.BgCyanHi      
gd.BgWhiteHi     

// Reset to default colors
gd.Reset 
```
***
## FONTS
```go
// Supported by SyncTerm
// usage: fmt.Fprint(gd.Output(), gd.Topaz)
gd.Mosoul        
gd.Potnoodle     
gd.Microknight    
gd.Microknightplus 
gd.Topaz          
gd.Topazplus      
gd.Ibm 
gd.Ibmthin         
```

***

## IDLE TIMER
```go
gd.Idle = 120 // seconds; 0 disables the timeout

// Optional: override what happens on timeout (default prints a notice and exits)
gd.IdleAction = func() {
    myCleanup()
    os.Exit(0)
}
```
> :point_up: Set the package-level `gd.Idle` (in seconds) before calling `gd.Pause`/`gd.Continue`. When the user idles at a prompt longer than this, `gd.IdleAction` runs — by default it prints a message and exits so the BBS can reclaim the node. Override `gd.IdleAction` to run your own cleanup/logging (or a non-terminating handler). Leave `gd.Idle` at `0` to disable — the timer is only armed when `gd.Idle > 0`.

## MISC
See the [source files](https://github.com/robbiew/godoors) — or run `go doc github.com/robbiew/godoors` — for other misc. functions.

## :clipboard: TO-DO
- ~~Time-out if no key press in X mins~~
- ~~Pop-up style window~~
- ~~Pause sequence (press any key to continue)~~
- ~~Confirm Y/n prompt~~
- ~~Get single key press from keyboard~~
- Get text input, max X characters
- ~~Idle/timeout timer example~~
- Write user data to text file
- Create a leader or score board
- Write user data to sqlite file
- Retrive/parse/display JSON data from the Internet APIs (16 colors, news, weather, etc.)
- Retrieve an ANSI file from the internet and display
- Add entry to end of log file
- Tidy on exit
- Save & Restore cursor position
- Create a scrollable/selectable list of things
- ANSI art file manipulation (scroll up/down, left/right)
- SIXEL support!
