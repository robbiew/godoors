//go:build !windows

package godoors

import (
	"fmt"
	"io"
)

// Unix BBS software supplies the caller through stdin/stdout even for type 2.
func platformConnection(info DropFile) (io.ReadWriteCloser, error) {
	if info.INI != nil && info.CommType == 2 {
		return nil, fmt.Errorf("godoors: DOORDROP.INI socket transport is currently supported only on Windows; use COMM_TYPE=stdio")
	}
	return nil, nil
}

// Preserve the historical metadata-only contract on Unix; its connection is
// already supplied by the BBS through stdin/stdout.
func sessionDropFile(path string) (DropFile, error) {
	if doorDropPath(path) != "" {
		return ReadDropFile(path)
	}
	alias, left, emulation, node, err := DropFileData(path)
	return DropFile{Alias: alias, TimeLeft: left, Emulation: emulation, Node: node}, err
}
