//go:build !windows

package godoors

import "io"

// Unix BBS software supplies the caller through stdin/stdout even for type 2.
func platformConnection(info DropFile) (io.ReadWriteCloser, error) { return nil, nil }

// Preserve the historical metadata-only contract on Unix; its connection is
// already supplied by the BBS through stdin/stdout.
func sessionDropFile(path string) (DropFile, error) {
	alias, left, emulation, node, err := DropFileData(path)
	return DropFile{Alias: alias, TimeLeft: left, Emulation: emulation, Node: node}, err
}
