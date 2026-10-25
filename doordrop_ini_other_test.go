//go:build !windows

package godoors

import "testing"

func TestDoorDropINISocketCannotFallBackToStdio(t *testing.T) {
	if _, err := platformConnection(DropFile{CommType: 2, Handle: 1234, INI: &DoorDropINI{}}); err == nil {
		t.Fatal("INI socket must not fall back to stdio")
	}
	if _, err := platformConnection(DropFile{CommType: 2, Handle: 1234}); err != nil {
		t.Fatalf("legacy Door32 behavior changed: %v", err)
	}
}
