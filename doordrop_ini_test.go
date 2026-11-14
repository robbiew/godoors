package godoors

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const iniFixture = "; DOORDROP.INI 1.0\r\n[file]\r\nFILE_TIME=2026-11-14T10:00:00-05:00\r\n[system]\r\nSYS_SOFTWARE=Test BBS 1.0\r\nSYS_NAME=Test\r\nSYS_OP=Sysop\r\nSYS_NODE_NUM=2\r\n[comm]\r\nCOMM_TYPE=stdio\r\nCOMM_CHARSET=CP437\r\n[user]\r\nUSER_ALIAS=Joe;Bob=#1\r\nUSER_NUMBER=42\r\nUSER_KEY=42.created\r\n[terminal]\r\nTERM_COLS=132\r\nTERM_ROWS=37\r\nTERM_TYPE=ansi\r\n[session]\r\nTIME_LEFT=61\r\nX_TEST_VALUE=kept\r\n"

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
	if info.Alias != "Joe;Bob=#1" || info.Emulation != 0 || info.TimeLeft != 0 || info.INI.TimeLeftSeconds != nil || info.INI.TermType != "dumb" || info.INI.Role != "user" {
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
		{"[file]", "[file]\r\nFILE_UTF8=2"},
		{"TERM_TYPE=ansi", ""},
		{"TERM_COLS=132", ""},
		{"TERM_ROWS=37", ""},
		{"TERM_COLS=132", "TERM_COLS=0"},
		{"TERM_ROWS=37", "TERM_ROWS=x"},
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
	if err != nil || info.Alias != "" || info.INI.Cols != 80 || info.INI.Rows != 24 || info.INI.TermType != "dumb" {
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

func TestDoorDropINITypedFields(t *testing.T) {
	info, err := ReadDropFile(writeINI(t, iniFixture+"USER_ROLE=sysop\r\n[preferences]\r\nPREF_SOUND=0\r\nPREF_PAUSE=7\r\nPREF_EXPERT=bad\r\n"))
	if err != nil {
		t.Fatal(err)
	}
	ini := info.INI
	if ini.Cols != 132 || ini.Rows != 37 || ini.TermType != "ansi" || ini.Role != "sysop" {
		t.Fatalf("got %+v", ini)
	}
	if ini.SoundAllowed() || !ini.Bool("PREF_PAUSE", false) || ini.Bool("PREF_EXPERT", false) || !ini.Bool("PREF_MOUSE", true) {
		t.Fatal("bool parsing")
	}
	var none *DoorDropINI
	if !none.SoundAllowed() {
		t.Fatal("nil INI must allow sound")
	}
	for _, role := range []string{"admin", "SYSOP"} {
		info, err := ReadDropFile(writeINI(t, iniFixture+"USER_ROLE="+role+"\r\n"))
		if err != nil || info.INI.Role != "user" {
			t.Fatalf("role %s: %+v %v", role, info.INI, err)
		}
	}
}

func TestDoorDropINIInitializeUsesHostSize(t *testing.T) {
	user, err := Initialize(writeINI(t, iniFixture))
	if err != nil {
		t.Fatal(err)
	}
	defer CloseSession()
	if user.W != 132 || user.H != 37 || user.ModalW != 132 || user.ModalH != 36 {
		t.Fatalf("got %+v", user)
	}
}

func TestPlayMusicHonorsPrefSound(t *testing.T) {
	for _, tc := range []struct {
		pref string
		want bool
	}{{"PREF_SOUND=0", false}, {"PREF_SOUND=1", true}, {"", true}} {
		t.Run(tc.pref, func(t *testing.T) {
			if _, err := OpenSession(writeINI(t, iniFixture+"[preferences]\r\n"+tc.pref+"\r\n")); err != nil {
				t.Fatal(err)
			}
			defer CloseSession()
			r, w, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			stdout := os.Stdout
			os.Stdout = w
			PlayMusic("cde")
			os.Stdout = stdout
			w.Close()
			out, _ := io.ReadAll(r)
			if got := len(out) > 0; got != tc.want {
				t.Fatalf("PlayMusic wrote %q", out)
			}
		})
	}
	CloseSession()
	if sessionINI() != nil {
		t.Fatal("CloseSession kept INI")
	}
}
