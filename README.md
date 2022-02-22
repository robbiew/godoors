# godoors

Helpful BBS "door" library in GO for creating linux-based applications that run over STDIO, using a terminal program like SyncTerm, MagiTerm, NetRunner or IGterm.

If you're not already running BBS software like Talisman, Mystic, Synchronet, Enigma 1/2 or WWIV (on linux), then this library probably isn't for you.

## Install
```go
go get github.com/robbiew/godoors
```

## General functions

***

### DROP FILES

```go 
godoors.DropFileData(path string) (string, int, int)
```

> Pass the path of the Door32.sys BBS drop file (including trailing slash), and itll return HANDLE/ALIAS, TIME LEFT (in minutes) and EMULATION TYPE (0 = Ascii, 1 = Ansi). You'll probably want to do this using a "startDoor.sh" file or from your BBS menu command (e.g. /path/to/drop/$NODE/Door32.sys/).

***
 
### TERMINAL HEIGHT AND WIDTH
```go
godoors.GetTermSize() (int, int)
```

> Tries to detect the user's terminal size. Returns HEIGHT and WIDTH. If it can't detect, it'll default to 25 and 80.

***
### DISPLAY ANSI ART
```go
godoors.PrintAnsi(file string)
```

> Pass the path of a valid ANSI art file and it'll strip the SAUCE record, then print line by line.

***
### DISPLAY ART AT X,Y COORDINATES
```go
godoors.PrintAnsiLoc(file string, x int, y int)
```

> Same as above, only it'll print the art to the screen at X/Y coordinates, incrementing the Y position after every line.

***

## Cursor related

```go
// Move the cursor n cells to up.
func CursorUp(n int) 

// Move the cursor n cells to down.
func CursorDown(n int) 

// Move the cursor n cells to right.
func CursorForward(n int) 

// Move the cursor n cells to left.
func CursorBack(n int) 

// Move cursor to beginning of the line n lines down.
func CursorNextLine(n int) 

// Move cursor to beginning of the line n lines up.
func CursorPreviousLine(n int) 

// Move cursor horizontally to x.
func CursorHorizontalAbsolute(x int) 

// Show the cursor.
func CursorShow() 

// Hide the cursor.
func CursorHide()
```
***
## Color
```go
// Text color  -- e.g. fmt.Println(godoors.Yellow)
Black         
Red          
Green         
Yellow     
Blue        
Magenta      
Cyan         
White         
BrightBlack   
BrightRed    
BrightGreen   
BrightYellow  
BrightBlue    
BrightMagenta 
BrightCyan    
BrightWhite   

// Background colors
BgBlack        
BgRed          
BgGreen        
BgYellow       
BgBlue          
BgMagenta       
BgCyan          
BgWhite         
BgBrightBlack   
BgBrightRed     
BgBrightGreen   
BgBrightYellow  
BgBrightBlue    
BgBrightMagenta 
BgBrightCyan    
BgBrightWhite   

// Reset to default colors
ColorReset 
```
***
### MISC
See ```godoors.go``` for other misc. functions.