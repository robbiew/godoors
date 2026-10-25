package godoors

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// DoorDropINI exposes experimental DOORDROP.INI draft 0.8 metadata. Text values
// retain the file's encoding (CP437 by default). UTF-8 files are not yet supported.
// Values includes unknown/vendor keys; optional keys absent from it retain their
// specification defaults. This API may change until the specification reaches 1.0.
type DoorDropINI struct {
	Values map[string]string
	// TimeLeftSeconds is nil when the host imposes no time limit. Unlike the
	// legacy TimeLeft minutes field, it distinguishes unlimited from expired.
	TimeLeftSeconds *int
}

// An explicit path takes precedence over the environment. Existing directory
// callers continue to select Door32, even if DOORDROP_INI is set.
func doorDropPath(path string) string {
	if path == "" {
		return os.Getenv("DOORDROP_INI")
	}
	if strings.EqualFold(filepath.Ext(path), ".ini") {
		return path
	}
	return ""
}

func readDoorDropINI(path string) (DropFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return DropFile{}, err
	}
	if end := bytes.IndexByte(data, 0x1a); end >= 0 {
		data = data[:end]
	}
	values := make(map[string]string)
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.Trim(scanner.Text(), " \t")
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "[") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key, value = strings.Trim(key, " \t"), strings.Trim(value, " \t")
		if _, exists := values[key]; !exists {
			values[key] = value
		}
	}
	if err := scanner.Err(); err != nil {
		return DropFile{}, fmt.Errorf("reading DOORDROP.INI: %w", err)
	}
	if values["FILE_UTF8"] == "1" {
		return DropFile{}, fmt.Errorf("godoors: experimental DOORDROP.INI does not support FILE_UTF8=1")
	}
	info := DropFile{Alias: values["USER_ALIAS"], INI: &DoorDropINI{Values: values}}
	number := func(key string) (int, error) {
		n, err := parseDoorDropNumber(values[key], 31)
		if err != nil {
			return 0, fmt.Errorf("DOORDROP.INI %s: %w", key, err)
		}
		return int(n), nil
	}
	if info.Node, err = number("SYS_NODE_NUM"); err != nil {
		return DropFile{}, err
	}
	if info.Node == 0 {
		return DropFile{}, fmt.Errorf("DOORDROP.INI SYS_NODE_NUM must be positive")
	}
	if _, present := values["TIME_LEFT"]; present {
		seconds, err := number("TIME_LEFT")
		if err != nil {
			return DropFile{}, err
		}
		info.INI.TimeLeftSeconds = &seconds
		info.TimeLeft = seconds / 60
		if seconds%60 != 0 {
			info.TimeLeft++
		}
	}
	switch values["TERM_TYPE"] {
	case "ansi", "avatar":
		info.Emulation = 1
	}
	switch values["COMM_TYPE"] {
	case "local", "stdio":
	case "socket":
		info.CommType = 2
		handle, err := parseDoorDropNumber(values["COMM_HANDLE"], strconv.IntSize)
		if err != nil {
			return DropFile{}, fmt.Errorf("DOORDROP.INI COMM_HANDLE: %w", err)
		}
		info.Handle = uintptr(handle)
	default:
		return DropFile{}, fmt.Errorf("godoors: unsupported DOORDROP.INI COMM_TYPE %q", values["COMM_TYPE"])
	}
	if values["COMM_TYPE"] != "local" {
		if info.Alias == "" {
			return DropFile{}, fmt.Errorf("DOORDROP.INI USER_ALIAS is required")
		}
		// GoDoors drawing and input are byte-oriented CP437. Do not silently send
		// CP437 graphics on a connection requiring a different encoding.
		if values["COMM_CHARSET"] != "CP437" {
			return DropFile{}, fmt.Errorf("godoors: unsupported DOORDROP.INI COMM_CHARSET %q", values["COMM_CHARSET"])
		}
	}
	return info, nil
}

func parseDoorDropNumber(value string, bits int) (uint64, error) {
	if value == "" || (len(value) > 1 && value[0] == '0') {
		return 0, fmt.Errorf("invalid unsigned decimal %q", value)
	}
	for _, c := range value {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("invalid unsigned decimal %q", value)
		}
	}
	return strconv.ParseUint(value, 10, bits)
}
