//go:build !windows

package godoors

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenSessionDoor32UnusedHandle(t *testing.T) {
	dir := t.TempDir()
	content := "2\n-1\n57600\nScorpio\n1\nReal Name\nAlias\n255\n58\n1\n2\n"
	if err := os.WriteFile(filepath.Join(dir, "door32.sys"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	info, err := OpenSession(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { CloseSession() })
	if info.Alias != "Alias" || info.TimeLeft != 58 || info.Emulation != 1 || info.Node != 2 {
		t.Fatalf("incorrect metadata: %+v", info)
	}
	if remoteSession() || Output() != os.Stdout {
		t.Fatal("Door32 stdio session selected a socket transport")
	}
	// Direct transport parsing stays strict for consumers that need a handle.
	if _, err := ReadDropFile(dir); err == nil {
		t.Fatal("ReadDropFile accepted an invalid unsigned handle")
	}
}
