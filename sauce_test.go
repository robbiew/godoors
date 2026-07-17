package godoors

import (
	"strings"
	"testing"
)

// makeSauce builds a valid 128-byte SAUCE record with the given comment count.
func makeSauce(comments byte) string {
	rec := make([]byte, 128)
	copy(rec, "SAUCE00")
	rec[104] = comments // TComments field
	return string(rec)
}

func TestTrimSauceNoRecordUnchanged(t *testing.T) {
	art := "line1\r\nline2\r\n"
	if got := TrimStringFromSauce(art); got != art {
		t.Fatalf("art without SAUCE was altered: %q", got)
	}
}

func TestTrimSauceStripsRecord(t *testing.T) {
	art := "HELLO ANSI ART\r\n"
	got := TrimStringFromSauce(art + "\x1a" + makeSauce(0))
	if got != art {
		t.Fatalf("expected %q, got %q", art, got)
	}
}

func TestTrimSauceStripsComntBlock(t *testing.T) {
	art := "art body\r\n"
	comnt := "COMNT" + strings.Repeat("x", 64) // one 64-byte comment line
	got := TrimStringFromSauce(art + comnt + makeSauce(1))
	if got != art {
		t.Fatalf("expected %q, got %q", art, got)
	}
}

func TestTrimSauceIgnoresMidStreamMarker(t *testing.T) {
	// "SAUCE00" appearing inside real art content must NOT trigger a trim.
	art := "prefix SAUCE00 not a real record\r\nmore art\r\n"
	if got := TrimStringFromSauce(art); got != art {
		t.Fatalf("mid-stream SAUCE00 caused truncation: %q", got)
	}
}
