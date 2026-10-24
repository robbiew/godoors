package godoors

import (
	"fmt"
	"io"
	"os"
)

// GoDoors has one active session per process. Initialize and CloseSession must
// not run concurrently with input, output, or terminal probes.
var activeConnection io.ReadWriteCloser

// Output returns the caller's output stream. Use fmt.Fprint(Output(), ...) for
// application text: fmt.Print still writes to the process's local stdout.
func Output() io.Writer {
	if activeConnection != nil {
		return activeConnection
	}
	return os.Stdout
}

func sessionInput() io.Reader {
	if activeConnection != nil {
		return activeConnection
	}
	return os.Stdin
}
func remoteSession() bool { return activeConnection != nil }

func resetSessionInput() {
	stdinPump, stdinReader = nil, nil
	probeMu.Lock()
	ctermKnown, ctermDetected = false, false
	probeMu.Unlock()
}

func connectSession(info DropFile) error {
	if activeConnection != nil {
		return fmt.Errorf("godoors: session already open; call CloseSession first")
	}
	conn, err := platformConnection(info)
	if err != nil {
		return err
	}
	if conn != nil {
		CloseInput()
		activeConnection = conn
		resetSessionInput()
	}
	return nil
}

// CloseSession restores local input and closes this process's inherited socket
// handle. It does not shut down the BBS's connection or close stdin/stdout.
// Call it explicitly before os.Exit, which does not run deferred cleanup.
func CloseSession() error {
	CloseInput()
	if activeConnection == nil {
		return nil
	}
	err := activeConnection.Close()
	activeConnection = nil
	resetSessionInput()
	return err
}
