package godoors

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDropFileDataParses(t *testing.T) {
	dir := t.TempDir()
	// door32.sys layout: lines are 0-indexed; alias=6, timeleft=8,
	// emulation=9, node=10.
	content := "2\n0\n57600\nMysticBBS\n1\nReal Name\nAlias\n255\n58\n1\n1\n"
	if err := os.WriteFile(filepath.Join(dir, "door32.sys"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	alias, timeLeft, emu, node, err := DropFileData(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if alias != "Alias" || timeLeft != 58 || emu != 1 || node != 1 {
		t.Fatalf("got (%q, %d, %d, %d)", alias, timeLeft, emu, node)
	}
}

func TestDropFileDataMissingReturnsError(t *testing.T) {
	if _, _, _, _, err := DropFileData(t.TempDir()); err == nil {
		t.Fatal("expected error for missing dropfile, got nil")
	}
}

func TestDropFileDataNonNumericReturnsError(t *testing.T) {
	dir := t.TempDir()
	content := "2\n0\n57600\nMysticBBS\n1\nReal Name\nAlias\n255\nNOTANUMBER\n1\n1\n"
	if err := os.WriteFile(filepath.Join(dir, "door32.sys"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, _, _, err := DropFileData(dir); err == nil {
		t.Fatal("expected error for non-numeric time-left, got nil")
	}
}
