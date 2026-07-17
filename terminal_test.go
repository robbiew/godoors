package godoors

import "testing"

func TestParseTermSizeReplyRejectsMalformedInput(t *testing.T) {
	if _, _, ok := parseTermSizeReply(";"); ok {
		t.Fatal("expected malformed reply to be rejected")
	}
}

func TestParseTermSizeReplyParsesValidInput(t *testing.T) {
	h, w, ok := parseTermSizeReply("\x1b[24;80R")
	if !ok {
		t.Fatal("expected valid reply to parse")
	}
	if h != 24 || w != 80 {
		t.Fatalf("got %d,%d", h, w)
	}
}