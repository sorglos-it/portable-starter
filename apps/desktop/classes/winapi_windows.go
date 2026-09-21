package classes

import (
	"errors"
	"io/fs"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"unsafe"
)

var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	user32   = syscall.NewLazyDLL("user32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	ole32    = syscall.NewLazyDLL("ole32.dll")

	procSetDefaultDllDirectories = kernel32.NewProc("SetDefaultDllDirectories")
	procGetUserDefaultUILanguage = kernel32.NewProc("GetUserDefaultUILanguage")
	procGetSystemDirectory       = kernel32.NewProc("GetSystemDirectoryW")
	procMessageBox               = user32.NewProc("MessageBoxW")
	procShellExecuteEx           = shell32.NewProc("ShellExecuteExW")
	procCoInitializeEx           = ole32.NewProc("CoInitializeEx")
)

// HardenDLLSearch lässt Windows Bibliotheken nur noch aus System32 laden. Im
// Programmordner liegen oft fremde DLLs, die der Starter nie laden darf.
// Muss vor allen anderen Aufrufen dieses Pakets laufen.
func HardenDLLSearch() {
	const loadLibrarySearchSystem32 = 0x00000800
	procSetDefaultDllDirectories.Call(loadLibrarySearchSystem32)
}

// UserLanguage liefert "de", wenn Windows auf Deutsch läuft, sonst "en".
func UserLanguage() string {
	id, _, _ := procGetUserDefaultUILanguage.Call()
	if id&0x3ff == 0x07 { // LANG_GERMAN
		return "de"
	}
	return "en"
}

// shellExecuteInfo entspricht SHELLEXECUTEINFOW.
type shellExecuteInfo struct {
	size       uint32
	mask       uint32
	hwnd       uintptr
	verb       *uint16
	file       *uint16
	parameters *uint16
	directory  *uint16
	show       int32
	instApp    uintptr
	idList     uintptr
	class      *uint16
	keyClass   uintptr
	hotKey     uint32
	icon       uintptr
	process    uintptr
}

// Launch startet exe mit dem Arbeitsordner dir wie ein Doppelklick im
// Explorer: Verlangt das Programm Adminrechte, fragt Windows nach,
// Konsolenprogramme bekommen ein Fenster. Bricht der Anwender die
// Windows-Abfrage ab, ist das kein Fehler.
func Launch(exe string, args []string, dir string) error {
	return shellExecute(exe, commandLine(args), dir)
}

// OpenInEditor öffnet path im Windows-Editor.
func OpenInEditor(path string) error {
	return shellExecute(filepath.Join(systemDir(), "notepad.exe"), commandLine([]string{path}), "")
}

func shellExecute(file, params, dir string) error {
	const (
		seeMaskNoAsync          = 0x00000100 // fertig starten, bevor der Starter endet
		seeMaskFlagNoUI         = 0x00000400 // Fehler meldet der Starter selbst
		coinitApartmentThreaded = 0x2
		coinitDisableOLE1DDE    = 0x4
		errorCancelled          = 1223
	)
	runtime.LockOSThread() // COM gilt je Thread
	defer runtime.UnlockOSThread()
	procCoInitializeEx.Call(0, coinitApartmentThreaded|coinitDisableOLE1DDE)

	info := shellExecuteInfo{mask: seeMaskNoAsync | seeMaskFlagNoUI, show: showCommand()}
	info.size = uint32(unsafe.Sizeof(info))
	var err error
	if info.file, err = syscall.UTF16PtrFromString(file); err != nil {
		return err
	}
	if info.parameters, err = syscall.UTF16PtrFromString(params); err != nil {
		return err
	}
	if dir != "" {
		if info.directory, err = syscall.UTF16PtrFromString(dir); err != nil {
			return err
		}
	}
	ok, _, callErr := procShellExecuteEx.Call(uintptr(unsafe.Pointer(&info)))
	if ok != 0 || callErr == syscall.Errno(errorCancelled) {
		return nil
	}
	return callErr
}

// commandLine setzt args zur Windows-Befehlszeile zusammen und maskiert dabei
// Leerzeichen und Anführungszeichen.
func commandLine(args []string) string {
	quoted := make([]string, len(args))
	for i, a := range args {
		quoted[i] = syscall.EscapeArg(a)
	}
	return strings.Join(quoted, " ")
}

// showCommand gibt weiter, wie der Starter selbst angezeigt werden soll, etwa
// „Minimiert“ aus einer Verknüpfung. Versteckt startet er nie.
func showCommand() int32 {
	const swHide, swShowNormal = 0, 1
	var si syscall.StartupInfo
	if syscall.GetStartupInfo(&si) == nil && si.Flags&syscall.STARTF_USESHOWWINDOW != 0 && si.ShowWindow != swHide {
		return int32(si.ShowWindow)
	}
	return swShowNormal
}

func systemDir() string {
	buf := make([]uint16, syscall.MAX_PATH)
	n, _, _ := procGetSystemDirectory.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if n == 0 || n >= uintptr(len(buf)) {
		return `C:\Windows\System32`
	}
	return syscall.UTF16ToString(buf[:n])
}

// describe liefert die Windows-Meldung zu err in der Sprache des Anwenders,
// ohne Go-Vorsatz wie „open C:\…:“.
func describe(err error) string {
	var errno syscall.Errno
	if errors.As(err, &errno) {
		const flags = syscall.FORMAT_MESSAGE_FROM_SYSTEM | syscall.FORMAT_MESSAGE_IGNORE_INSERTS
		buf := make([]uint16, 512)
		if n, e := syscall.FormatMessage(flags, 0, uint32(errno), 0, buf, nil); e == nil && n > 0 {
			return strings.TrimSpace(syscall.UTF16ToString(buf[:n]))
		}
	}
	var pathErr *fs.PathError
	if errors.As(err, &pathErr) {
		return pathErr.Err.Error()
	}
	return err.Error()
}

const (
	mbOK            = 0x00000000
	mbYesNo         = 0x00000004
	mbIconError     = 0x00000010
	mbIconWarning   = 0x00000030
	mbSetForeground = 0x00010000
	idYes           = 6
)

// ShowError zeigt text als Fehlermeldung.
func ShowError(title, text string) {
	showMessage(title, text, mbOK|mbIconError)
}

// Ask zeigt text mit Ja und Nein und meldet, ob der Anwender Ja gewählt hat.
func Ask(title, text string) bool {
	return showMessage(title, text, mbYesNo|mbIconWarning) == idYes
}

func showMessage(title, text string, style uintptr) uintptr {
	t, _ := syscall.UTF16PtrFromString(strings.ReplaceAll(title, "\x00", ""))
	m, _ := syscall.UTF16PtrFromString(strings.ReplaceAll(text, "\x00", ""))
	r, _, _ := procMessageBox.Call(0, uintptr(unsafe.Pointer(m)), uintptr(unsafe.Pointer(t)), style|mbSetForeground)
	return r
}
