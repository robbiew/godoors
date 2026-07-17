package godoors

import (
	"fmt"
	"strings"
)

// Credit to @richorr
func PipeColorToEscCode(ansiColor string) (string, bool) {
	if len(strings.Trim(ansiColor, " ")) < 2 {
		return "", false
	}
	tint := ""
	isColor := true

	switch ansiColor[0] {
	case '0':
		tint = BgBlack
	case '1':
		tint = BgBlue
	case '2':
		tint = BgGreen
	case '3':
		tint = BgCyan
	case '4':
		tint = BgRed
	case '5':
		tint = BgMagenta
	case '6':
		tint = BgYellow
	case '7':
		tint = BgWhite
	case '8':
		tint = BgBlackHi
	case '9':
		tint = BgBlueHi
	case 'A':
		tint = BgGreenHi
	case 'B':
		tint = BgCyanHi
	case 'C':
		tint = BgRedHi
	case 'D':
		tint = BgMagentaHi
	case 'E':
		tint = BgYellowHi
	case 'F':
		tint = BgWhiteHi
	default:
		isColor = false
	}

	if isColor {
		switch ansiColor[1] {
		case '0':
			tint += Black
		case '1':
			tint += Blue
		case '2':
			tint += Green
		case '3':
			tint += Cyan
		case '4':
			tint += Red
		case '5':
			tint += Magenta
		case '6':
			tint += Yellow
		case '7':
			tint += White
		case '8':
			tint += BlackHi
		case '9':
			tint += BlueHi
		case 'A':
			tint += GreenHi
		case 'B':
			tint += CyanHi
		case 'C':
			tint += RedHi
		case 'D':
			tint += MagentaHi
		case 'E':
			tint += YellowHi
		case 'F':
			tint += WhiteHi
		default:
			isColor = false
		}
	}
	return tint, isColor
}

func PrintPipeColor(text string, defaultTint string) string {
	tint := defaultTint

	coloredSections := strings.Split(text, "|")
	for i, ansiBlock := range coloredSections {
		if len(strings.Trim(ansiBlock, " ")) < 2 {
			if i == 0 {
				fmt.Print(tint, ansiBlock, Reset)
			} else {
				fmt.Print(tint, "|", ansiBlock, Reset)
			}
		} else {
			newTint, isColor := PipeColorToEscCode(ansiBlock[0:2])

			if isColor {
				// Change the tint and then print the following block
				tint = newTint
				fmt.Print(tint, ansiBlock[2:], Reset)
			} else {
				// Print the pipe if a color was not found
				if i == 0 {
					fmt.Print(tint, ansiBlock, Reset)
				} else {
					fmt.Print(tint, "|", ansiBlock, Reset)
				}
			}
		}
	}
	return tint
}
