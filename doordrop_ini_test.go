package godoors

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const iniFixture = "; draft 0.8\r\n[file]\r\n[system]\r\nSYS_NODE_NUM=2\r\n[comm]\r\nCOMM_TYPE=stdio\r\nCOMM_CHARSET=CP437\r\n[user]\r\nUSER_ALIAS=Joe;Bob=#1\r\nUSER_NUMBER=42\r\nUSER_KEY=42.created\r\n[terminal]\r\nTERM_TYPE=ansi\r\n[session]\r\nTIME_LEFT=61\r\nX_TEST_VALUE=kept\r\n"

func writeINI(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "custom name.ini")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestDoorDropINIEntryPoints(t *testing.T) {
	for _, newline := range []string{"\r\n", "\n"} {
		t.Run(newline, func(t *testing.T) {
			path := writeINI(t, strings.ReplaceAll(iniFixture, "\r\n", newline))
			info, err := ReadDropFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if info.Alias != "Joe;Bob=#1" || info.TimeLeft != 2 || info.Node != 2 || info.Emulation != 1 || info.INI == nil || *info.INI.TimeLeftSeconds != 61 || info.INI.Values["X_TEST_VALUE"] != "kept" {
				t.Fatalf("got %+v", info)
			}
			alias, left, emu, node, err := DropFileData(path)
			if err != nil || alias != info.Alias || left != 2 || emu != 1 || node != 2 {
				t.Fatalf("metadata: %q %d %d %d %v", alias, left, emu, node, err)
			}
			session, err := OpenSession(path)
			if err != nil {
				t.Fatal(err)
			}
			defer CloseSession()
			if session.INI == nil {
				t.Fatal("session lost INI metadata")
			}
		})
	}
}

func TestDoorDropINIDiscovery(t *testing.T) {
	path := writeINI(t, iniFixture)
	t.Setenv("DOORDROP_INI", path)
	if _, err := ReadDropFile(""); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	writeFixture(t, dir, "DOOR32.SYS")
	info, err := ReadDropFile(dir)
	if err != nil || info.INI != nil || info.Alias != "Alias" {
		t.Fatalf("Door32 precedence: %+v %v", info, err)
	}
	explicit := writeINI(t, strings.ReplaceAll(iniFixture, "Joe;Bob=#1", "Explicit"))
	info, err = ReadDropFile(explicit)
	if err != nil || info.Alias != "Explicit" {
		t.Fatalf("explicit precedence: %+v %v", info, err)
	}
}

func TestDoorDropINIDefaultsAndParsing(t *testing.T) {
	content := strings.ReplaceAll(iniFixture, "TIME_LEFT=61\r\n", "")
	content = strings.ReplaceAll(content, "TERM_TYPE=ansi", "TERM_TYPE=future")
	content += "USER_ALIAS=second\r\nignored line\r\n\x1aSYS_NODE_NUM=bad\r\n"
	info, err := ReadDropFile(writeINI(t, content))
	if err != nil {
		t.Fatal(err)
	}
	if info.Alias != "Joe;Bob=#1" || info.Emulation != 0 || info.TimeLeft != 0 || info.INI.TimeLeftSeconds != nil {
		t.Fatalf("got %+v", info)
	}
	for _, seconds := range []string{"0", "1", "60", "2147483647"} {
		info, err := ReadDropFile(writeINI(t, strings.ReplaceAll(iniFixture, "TIME_LEFT=61", "TIME_LEFT="+seconds)))
		if err != nil || info.INI.TimeLeftSeconds == nil {
			t.Fatalf("seconds=%s: %+v %v", seconds, info, err)
		}
	}
}

func TestDoorDropINIInvalidValues(t *testing.T) {
	for _, replacement := range []struct{ from, to string }{
		{"SYS_NODE_NUM=2", "SYS_NODE_NUM=0"},
		{"SYS_NODE_NUM=2", "SYS_NODE_NUM=bad"},
		{"TIME_LEFT=61", "TIME_LEFT=-1"},
		{"TIME_LEFT=61", "TIME_LEFT=+1"},
		{"TIME_LEFT=61", "TIME_LEFT=01"},
		{"TIME_LEFT=61", "TIME_LEFT=2147483648"},
		{"COMM_TYPE=stdio", "COMM_TYPE=telnet"},
		{"COMM_TYPE=stdio", "COMM_TYPE=serial"},
		{"COMM_TYPE=stdio", "COMM_TYPE=fossil"},
		{"COMM_TYPE=stdio", "COMM_TYPE=uart"},
		{"COMM_TYPE=stdio", "COMM_TYPE=future"},
		{"COMM_TYPE=stdio", "COMM_TYPE=socket"},
		{"COMM_CHARSET=CP437", "COMM_CHARSET=UTF-8"},
		{"USER_ALIAS=Joe;Bob=#1", "USER_ALIAS="},
		{"[file]", "[file]\r\nFILE_UTF8=1"},
	} {
		t.Run(replacement.to, func(t *testing.T) {
			if _, err := ReadDropFile(writeINI(t, strings.ReplaceAll(iniFixture, replacement.from, replacement.to))); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestDoorDropINILocalAndSocketMetadata(t *testing.T) {
	info, err := ReadDropFile(writeINI(t, "COMM_TYPE=local\nSYS_NODE_NUM=1\n"))
	if err != nil || info.Alias != "" {
		t.Fatalf("local: %+v %v", info, err)
	}
	info, err = ReadDropFile(writeINI(t, strings.ReplaceAll(iniFixture, "COMM_TYPE=stdio", "COMM_TYPE=socket\r\nCOMM_HANDLE=1234")))
	if err != nil || info.Handle != 1234 || info.CommType != 2 {
		t.Fatalf("socket: %+v %v", info, err)
	}
}

func TestDoorDropINISessionMetadata(t *testing.T) {
	info, err := sessionDropFile(writeINI(t, iniFixture))
	if err != nil || info.INI == nil || info.Node != 2 {
		t.Fatalf("Initialize metadata path: %+v %v", info, err)
	}
}

func TestDoor32DirectoryWithINIExtension(t *testing.T) {
	t.Setenv("DOORDROP_INI", writeINI(t, iniFixture))
	for _, name := range []string{"node.ini", "Node.INI"} {
		dir := filepath.Join(t.TempDir(), name)
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
		writeFixture(t, dir, "DOOR32.SYS")
		for _, path := range []string{dir, dir + string(os.PathSeparator)} {
			t.Run(name+"/"+filepath.Base(path), func(t *testing.T) {
				info, err := ReadDropFile(path)
				if err != nil || info.INI != nil || info.Alias != "Alias" {
					t.Fatalf("ReadDropFile: %+v %v", info, err)
				}
				alias, left, emu, node, err := DropFileData(path)
				if err != nil || alias != "Alias" || left != 58 || emu != 1 || node != 1 {
					t.Fatalf("DropFileData: %q %d %d %d %v", alias, left, emu, node, err)
				}
				info, err = sessionDropFile(path)
				if err != nil || info.INI != nil || info.Alias != "Alias" {
					t.Fatalf("Initialize metadata: %+v %v", info, err)
				}
				info, err = OpenSession(path)
				if err != nil {
					t.Fatal(err)
				}
				defer CloseSession()
				if info.INI != nil || info.Alias != "Alias" {
					t.Fatalf("OpenSession: %+v", info)
				}
			})
		}
	}
}
