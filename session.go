package godoors

import (
	"fmt"
	"io"
	"os"
	"sync"
)

// GoDoors has one active session per process. Initialize and CloseSession must
// not run concurrently with input or terminal probes. Output lookup is safe
// during closure; a previously returned writer may report a closed connection.
var activeConnection io.ReadWriteCloser
var connectionMu sync.RWMutex
var lifecycleMu sync.Mutex

// Output returns the caller's output stream. Use fmt.Fprint(Output(), ...) for
// application text: fmt.Print still writes to the process's local stdout.
func Output() io.Writer {
	connectionMu.RLock()
	defer connectionMu.RUnlock()
	if activeConnection != nil {
		return activeConnection
	}
	return os.Stdout
}

func sessionInput() io.Reader {
	connectionMu.RLock()
	defer connectionMu.RUnlock()
	if activeConnection != nil {
		return activeConnection
	}
	return os.Stdin
}
func remoteSession() bool {
	connectionMu.RLock()
	defer connectionMu.RUnlock()
	return activeConnection != nil
}

func resetSessionInput() {
	stdinPump, stdinReader = nil, nil
	crReader = nil
	probeMu.Lock()
	ctermKnown, ctermDetected = false, false
	probeMu.Unlock()
}

func connectSession(info DropFile) error {
	lifecycleMu.Lock()
	defer lifecycleMu.Unlock()
	if remoteSession() {
		return fmt.Errorf("godoors: session already open; call CloseSession first")
	}
	conn, err := platformConnection(info)
	if err != nil {
		return err
	}
	if conn != nil {
		CloseInput()
		connectionMu.Lock()
		activeConnection = conn
		connectionMu.Unlock()
		resetSessionInput()
	}
	return nil
}

// CloseSession restores local input and closes this process's inherited socket
// handle. It does not shut down the BBS's connection or close stdin/stdout.
// Call it explicitly before os.Exit, which does not run deferred cleanup.
func CloseSession() error {
	lifecycleMu.Lock()
	defer lifecycleMu.Unlock()
	CloseInput()
	if !remoteSession() {
		return nil
	}
	if err := closeConnection(); err != nil {
		return err
	}
	resetSessionInput()
	return nil
}

// Caller holds lifecycleMu. Output may run concurrently; only remove the
// connection after a successful close so failed closes remain retryable.
func closeConnection() error {
	connectionMu.RLock()
	conn := activeConnection
	connectionMu.RUnlock()
	if conn == nil {
		return nil
	}
	if err := conn.Close(); err != nil {
		return err
	}
	connectionMu.Lock()
	activeConnection = nil
	connectionMu.Unlock()
	return nil
}
