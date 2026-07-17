package godoors

import "strings"

// SAUCE record layout constants. See http://www.acid.org/info/sauce/sauce.htm
const (
	sauceRecordLen = 128 // the trailer is always exactly 128 bytes
	sauceID        = "SAUCE00"
	sauceComments  = 104 // offset of the 1-byte comment-line count
	comntID        = "COMNT"
	comntLineLen   = 64 // each COMNT comment line is 64 bytes
)

// TrimStringFromSauce removes a trailing SAUCE metadata record — plus any
// COMNT comment block and DOS EOF (Ctrl-Z) marker preceding it — from ANSI or
// ASCII art. SAUCE is a fixed 128-byte trailer anchored at the very end of the
// file, so it is only stripped when found there; art whose visible content
// happens to contain the bytes "SAUCE00" or "COMNT" is left untouched.
func TrimStringFromSauce(s string) string {
	if len(s) < sauceRecordLen {
		return s
	}
	record := s[len(s)-sauceRecordLen:]
	if !strings.HasPrefix(record, sauceID) {
		return s
	}
	body := s[:len(s)-sauceRecordLen]

	// A COMNT block, when present, sits immediately before the SAUCE record.
	// Its length is fixed by the record's comment-line count, so we can locate
	// it exactly instead of guessing.
	if n := int(record[sauceComments]); n > 0 {
		comntLen := len(comntID) + n*comntLineLen
		if len(body) >= comntLen && strings.HasPrefix(body[len(body)-comntLen:], comntID) {
			body = body[:len(body)-comntLen]
		}
	}

	// Drop a trailing DOS EOF marker that conventionally precedes SAUCE data.
	return strings.TrimRight(body, "\x1a")
}
