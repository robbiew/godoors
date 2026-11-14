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

// DoorDropINI exposes DOORDROP.INI 1.0 metadata. Text values retain the file's
// encoding, which GoDoors requires to be CP437: a FILE_UTF8 file is rejected.
// Values holds every first-occurrence key, including unknown and vendor keys,
// exactly as written; optional keys absent from it take their specification
// defaults, which the typed fields and Bool apply.
type DoorDropINI struct {
	Values map[string]string
	// TimeLeftSeconds is nil when the host imposes no time limit. Unlike the
	// legacy TimeLeft minutes field, it distinguishes unlimited from expired.
	TimeLeftSeconds *int
	// TermType is TERM_TYPE: dumb, ansi, avatar, petscii or atascii. A token
	// GoDoors does not recognize reads as dumb, as the specification requires.
	TermType string
	// Cols and Rows are TERM_COLS and TERM_ROWS, the size the host itself uses
	// for the session. They are required unless COMM_TYPE is local, where a
	// missing value defaults to 80 by 24.
	Cols, Rows int
	// Role is USER_ROLE: user, cosysop, sysop or guest. Missing or unknown
	// values read as user. It is not authentication.
	Role string
}

// Bool reads a DOORDROP.INI bool key: 0 is false and any other unsigned
// decimal is true. A missing or malformed value returns def, the key's
// specification default (for example PREF_SOUND defaults to true).
func (d *DoorDropINI) Bool(key string, def bool) bool {
	if d == nil {
		return def
	}
	n, err := parseDoorDropNumber(d.Values[key], 64)
	if err != nil {
		return def
	}
	return n != 0
}

// SoundAllowed reports whether the user permits audio (PREF_SOUND). A nil
// receiver, as for a Door32 session, allows sound.
func (d *DoorDropINI) SoundAllowed() bool { return d.Bool("PREF_SOUND", true) }

// An explicit path takes precedence over the environment. Existing directory
// callers continue to select Door32, even if DOORDROP_INI is set.
func doorDropPath(path string) string {
	if path == "" {
		return os.Getenv("DOORDROP_INI")
	}
	if strings.EqualFold(filepath.Ext(path), ".ini") {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			return ""
		}
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
	ini := &DoorDropINI{Values: values}
	// A door that can't decode UTF-8 MUST reject the file. Any true bool counts.
	if ini.Bool("FILE_UTF8", false) {
		return DropFile{}, fmt.Errorf("godoors: DOORDROP.INI FILE_UTF8 text is not supported; configure the door for CP437")
	}
	info := DropFile{Alias: values["USER_ALIAS"], INI: ini}
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
	local := values["COMM_TYPE"] == "local"
	switch ini.TermType = values["TERM_TYPE"]; ini.TermType {
	case "ansi", "avatar":
		info.Emulation = 1
	case "dumb", "petscii", "atascii":
	case "":
		if !local {
			return DropFile{}, fmt.Errorf("DOORDROP.INI TERM_TYPE is required")
		}
		ini.TermType = "dumb"
	default:
		ini.TermType = "dumb"
	}
	for _, size := range []struct {
		key  string
		dest *int
		def  int
	}{{"TERM_COLS", &ini.Cols, 80}, {"TERM_ROWS", &ini.Rows, 24}} {
		if _, present := values[size.key]; !present && local {
			*size.dest = size.def
			continue
		}
		n, err := number(size.key)
		if err != nil {
			return DropFile{}, err
		}
		if n == 0 {
			return DropFile{}, fmt.Errorf("DOORDROP.INI %s must be positive", size.key)
		}
		*size.dest = n
	}
	switch ini.Role = values["USER_ROLE"]; ini.Role {
	case "user", "cosysop", "sysop", "guest":
	default:
		ini.Role = "user"
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
	if !local {
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
