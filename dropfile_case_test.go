package godoors

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFixture(t *testing.T, dir, name string) {
	t.Helper()
	content := "0\n0\n57600\nTest BBS\n1\nReal Name\nAlias\n255\n58\n1\n1\n"
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestOpenDropFileAnyCase(t *testing.T) {
	for _, name := range []string{"door32.sys", "DOOR32.SYS", "Door32.Sys"} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			writeFixture(t, dir, name)
			f, err := openDropFile(dir)
			if err != nil {
				t.Fatalf("openDropFile failed for %s: %v", name, err)
			}
			f.Close()
		})
	}
}

func TestOpenDropFileTrailingSlashOptional(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "door32.sys")
	for _, p := range []string{dir, dir + string(os.PathSeparator)} {
		if f, err := openDropFile(p); err != nil {
			t.Fatalf("openDropFile(%q) failed: %v", p, err)
		} else {
			f.Close()
		}
	}
}

func TestOpenDropFileMissing(t *testing.T) {
	if f, err := openDropFile(t.TempDir()); err == nil {
		f.Close()
		t.Fatal("expected error for missing dropfile")
	}
}

func TestPathDirCaseIsPreserved(t *testing.T) {
	// The old implementation lowercased the WHOLE path, breaking
	// mixed-case directories on case-sensitive filesystems.
	base := t.TempDir()
	dir := filepath.Join(base, "NodeOne")
	if err := os.Mkdir(dir, 0755); err != nil {
		t.Fatal(err)
	}
	writeFixture(t, dir, "door32.sys")
	f, err := openDropFile(dir)
	if err != nil {
		t.Fatalf("mixed-case directory broke dropfile open: %v", err)
	}
	f.Close()
}
