package classes

import (
	"testing"
	"unsafe"
)

func TestCommandLine(t *testing.T) {
	got := commandLine([]string{"--user-data-dir=C:\\Mein Ordner\\daten", `say "hi"`, "", "plain"})
	want := `"--user-data-dir=C:\Mein Ordner\daten" "say \"hi\"" "" plain`
	if got != want {
		t.Errorf("%s\nerwartet\n%s", got, want)
	}
}

func TestShellExecuteInfoSize(t *testing.T) {
	// SHELLEXECUTEINFOW ist unter 64-Bit-Windows 112 Bytes groß.
	if size := unsafe.Sizeof(shellExecuteInfo{}); size != 112 {
		t.Errorf("SHELLEXECUTEINFOW %d Bytes, erwartet 112", size)
	}
}
