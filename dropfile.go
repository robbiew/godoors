package godoors

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// openDropFile opens door32.sys in dir regardless of the filename's case
// (BBS packages variously write door32.sys, DOOR32.SYS, Door32.Sys). The
// directory's own case is preserved. Trailing slash on dir is optional.
func openDropFile(dir string) (*os.File, error) {
	f, err := os.Open(filepath.Join(dir, "door32.sys"))
	if err == nil {
		return f, nil
	}
	entries, dirErr := os.ReadDir(dir)
	if dirErr != nil {
		// The directory itself couldn't be read; surface that alongside the
		// original open failure instead of masking it.
		return nil, errors.Join(err, dirErr)
	}
	for _, e := range entries {
		if !e.IsDir() && strings.EqualFold(e.Name(), "door32.sys") {
			return os.Open(filepath.Join(dir, e.Name()))
		}
	}
	return nil, err
}

// DropFileData reads a door32.sys drop file from the given directory and
// returns the user's alias, time left (minutes), emulation type (0 = ASCII,
// 1 = ANSI) and node number. Any I/O or parse failure is returned as an error
// so the caller can decide how to handle it, rather than terminating the host.
func DropFileData(path string) (alias string, timeLeft, emulation, node int, err error) {
	file, err := openDropFile(path)
	if err != nil {
		return "", 0, 0, 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	var text []string
	for scanner.Scan() {
		text = append(text, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return "", 0, 0, 0, fmt.Errorf("reading drop file: %w", err)
	}

	// door32.sys is line-oriented; the fields we need live at fixed indexes.
	const (
		aliasLine     = 6
		timeLeftLine  = 8
		emulationLine = 9
		nodeLine      = 10
	)
	if len(text) <= nodeLine {
		return "", 0, 0, 0, fmt.Errorf("drop file has %d lines, need at least %d", len(text), nodeLine+1)
	}

	alias = text[aliasLine]
	if timeLeft, err = strconv.Atoi(text[timeLeftLine]); err != nil {
		return "", 0, 0, 0, fmt.Errorf("parsing time left: %w", err)
	}
	if emulation, err = strconv.Atoi(text[emulationLine]); err != nil {
		return "", 0, 0, 0, fmt.Errorf("parsing emulation: %w", err)
	}
	if node, err = strconv.Atoi(text[nodeLine]); err != nil {
		return "", 0, 0, 0, fmt.Errorf("parsing node number: %w", err)
	}
	return alias, timeLeft, emulation, node, nil
}
