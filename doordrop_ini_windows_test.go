//go:build windows

package godoors

import "testing"

func TestDoorDropINISocketCannotFallBackToStdio(t *testing.T) {
	for _, handle := range []uintptr{0, ^uintptr(0)} {
		if _, err := platformConnection(DropFile{CommType: 2, Handle: handle, INI: &DoorDropINI{}}); err == nil {
			t.Fatalf("INI handle %d must not fall back to stdio", handle)
		}
	}
}
