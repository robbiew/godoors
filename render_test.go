package godoors

import (
	"io"
	"os"
	"strings"
	"testing"
)

// captureStdout redirects os.Stdout for the duration of fn and returns
// everything written to it.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	done := make(chan string)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()
	fn()
	w.Close()
	os.Stdout = orig
	return <-done
}

// Art and user text routinely contain '%'. When such data is passed as a
// printf format string, Go mangles it into "%!x(MISSING)" verbs. These
// functions must emit the data verbatim.
func TestPrintStringLocPassesPercentThrough(t *testing.T) {
	out := captureStdout(t, func() {
		PrintStringLoc("100% done", 1, 1)
	})
	if !strings.Contains(out, "100% done") {
		t.Fatalf("percent sign mangled: %q", out)
	}
	if strings.Contains(out, "MISSING") {
		t.Fatalf("format verb leaked into output: %q", out)
	}
}

func TestPrintAnsiPassesPercentThrough(t *testing.T) {
	out := captureStdout(t, func() {
		PrintAnsi("50% | 3d art %s here", 0, 100)
	})
	if !strings.Contains(out, "50% |") || !strings.Contains(out, "%s here") {
		t.Fatalf("art content mangled: %q", out)
	}
	if strings.Contains(out, "MISSING") {
		t.Fatalf("format verb leaked into output: %q", out)
	}
}
