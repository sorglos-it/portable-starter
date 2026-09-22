// build baut den Portable Starter; aufgerufen von tools\build.bat.
//
//	go run .              dist\starter.exe mit Standard-Icon und dist\config_starter.json
//	go run . Ordner …     je Programmordner portable_<programm>.exe mit Icon und
//	                      Namen des Programms, das der Starter dort findet; fehlt
//	                      config_starter.json, kommt das Beispiel dazu. Eine Kopie
//	                      jedes Starters landet in dist\
package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"starter/classes"
)

// Pfade ab tools\, dem Arbeitsordner von go run.
const (
	app         = "../apps/desktop"
	syso        = app + "/rsrc_windows_amd64.syso"
	defaultIcon = app + "/assets/icon.ico"
	example     = app + "/config/config_starter.json.example"
	dist        = "../dist"
)

func main() {
	texts := classes.LoadTexts(os.DirFS(app), classes.UserLanguage())
	failed := false
	fail := func(where string, err error) {
		fmt.Fprintf(os.Stderr, "FEHLER %s: %s\n", where, texts.Message(err))
		failed = true
	}

	raw, err := os.ReadFile(app + "/VERSION")
	if err != nil {
		fail("VERSION", err)
		os.Exit(1)
	}
	version, err := parseVersion(string(raw))
	if err != nil {
		fail("VERSION", err)
		os.Exit(1)
	}
	cfg, err := os.ReadFile(example)
	if err != nil {
		fail("Beispiel-Konfiguration", err)
		os.Exit(1)
	}

	if len(os.Args) < 2 {
		if err := buildDefault(version, cfg); err != nil {
			fail("dist", err)
		}
	}
	var built []string
	for _, folder := range os.Args[1:] {
		if !filepath.IsAbs(folder) {
			folder = filepath.Join(os.Getenv("STARTER_CWD"), folder) // Ordner relativ zum Aufruf von build.bat
		}
		file, err := buildFor(folder, version, cfg)
		if err != nil {
			fail(folder, err)
			continue
		}
		built = append(built, file)
	}
	// Kopien erst nach allen Builds: dist\ liegt im Repo, ein geänderter
	// Stand dort stempelte die folgenden Builds als „geändert“ (vcs.modified).
	if err := copyToDist(built); err != nil {
		fail("dist", err)
	}
	// go build und go test finden danach wieder das Standard-Icon vor
	if err := writeDefaultSyso(version, "starter.exe"); err != nil {
		fail("syso", err)
	}
	if failed {
		os.Exit(1)
	}
}

func writeDefaultSyso(version [4]uint16, filename string) error {
	images, err := loadIcon(defaultIcon)
	if err != nil {
		return err
	}
	return writeSyso(resInfo{images, version, "Portable Starter", filename}, syso)
}

// buildDefault baut den allgemeinen Starter mit Standard-Icon nach dist\.
func buildDefault(version [4]uint16, cfg []byte) error {
	if err := os.MkdirAll(dist, 0o755); err != nil {
		return err
	}
	if err := writeDefaultSyso(version, "starter.exe"); err != nil {
		return err
	}
	if err := goBuild(dist + "/starter.exe"); err != nil {
		return err
	}
	if err := os.WriteFile(dist+"/config_starter.json", cfg, 0o644); err != nil {
		return err
	}
	fmt.Println(`dist\starter.exe und dist\config_starter.json`)
	return nil
}

// buildFor legt in folder portable_<programm>.exe mit Icon und Namen des
// Programms ab, das der Starter dort starten wird – neben ihm oder in app\ –,
// und liefert den Pfad der EXE.
func buildFor(folder string, version [4]uint16, example []byte) (string, error) {
	cfgPath := filepath.Join(folder, classes.ConfigName)
	data, err := os.ReadFile(cfgPath)
	missing := errors.Is(err, fs.ErrNotExist)
	if missing {
		data = example
	} else if err != nil {
		return "", err
	}
	cfg, err := classes.ParseConfig(data)
	if err != nil {
		return "", err
	}
	target, err := cfg.Find(folder, "")
	if err != nil {
		return "", err
	}

	images, source, err := iconFor(folder, target.Exe)
	if err != nil {
		return "", err
	}
	name := productName(target.Exe)
	if name == "" {
		name = filepath.Base(folder)
	}
	file := "portable_" + slug(target.Exe) + ".exe"
	if err := writeSyso(resInfo{images, version, name + " portable", file}, syso); err != nil {
		return "", err
	}
	out := filepath.Join(folder, file)
	if err := goBuild(out); err != nil {
		return "", err
	}
	if missing {
		if err := os.WriteFile(cfgPath, example, 0o644); err != nil {
			return "", err
		}
	}
	fmt.Printf("%s  ->  %s  („%s portable“, Icon: %s)\n", folder, file, name, source)
	return out, nil
}

// copyToDist legt eine Kopie der gebauten Starter in dist\ ab.
func copyToDist(files []string) error {
	if len(files) == 0 {
		return nil
	}
	if err := os.MkdirAll(dist, 0o755); err != nil {
		return err
	}
	names := make([]string, len(files))
	for i, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		names[i] = filepath.Base(file)
		if err := os.WriteFile(filepath.Join(dist, names[i]), data, 0o755); err != nil {
			return err
		}
	}
	fmt.Printf("Kopie in dist\\: %s\n", strings.Join(names, ", "))
	return nil
}

// iconFor liefert das Icon des Programms: aus seiner EXE, sonst aus einer
// gleichnamigen .ico-Datei im Programmordner (QElectroTech legt sie nach ico\),
// sonst das Standard-Icon. source sagt, woher es kommt.
func iconFor(folder, exe string) (images []iconImage, source string, err error) {
	if images, err := loadIcon(exe); err == nil {
		return images, filepath.Base(exe), nil
	}
	want := strings.TrimSuffix(strings.ToLower(filepath.Base(exe)), ".exe") + ".ico"
	var found string
	filepath.WalkDir(folder, func(path string, d fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return nil
		case d.IsDir() && strings.Count(path[len(folder):], string(filepath.Separator)) > 3:
			return filepath.SkipDir
		case !d.IsDir() && strings.EqualFold(d.Name(), want):
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	if found != "" {
		if images, err := loadIcon(found); err == nil {
			rel, _ := filepath.Rel(folder, found)
			return images, rel, nil
		}
	}
	images, err = loadIcon(defaultIcon)
	return images, "Standard-Icon", err
}

func goBuild(out string) error {
	abs, err := filepath.Abs(out)
	if err != nil {
		return err
	}
	cmd := exec.Command("go", "build", "-trimpath", "-ldflags", "-s -w -H windowsgui", "-o", abs, ".")
	cmd.Dir = app
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	return cmd.Run()
}

// slug macht aus „bin/draw.io.exe“ den Namensteil „drawio“.
func slug(exe string) string {
	base := strings.ToLower(filepath.Base(filepath.FromSlash(exe)))
	var b strings.Builder
	for _, r := range strings.TrimSuffix(base, ".exe") {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "programm"
	}
	return b.String()
}

var (
	versionDLL                 = syscall.NewLazyDLL("version.dll")
	procGetFileVersionInfoSize = versionDLL.NewProc("GetFileVersionInfoSizeW")
	procGetFileVersionInfo     = versionDLL.NewProc("GetFileVersionInfoW")
	procVerQueryValue          = versionDLL.NewProc("VerQueryValueW")
)

// productName liest den Programmnamen aus den Versionsangaben der EXE:
// ProductName, ersatzweise FileDescription; leer, wenn sie keine hat.
func productName(exe string) string {
	path, err := syscall.UTF16PtrFromString(exe)
	if err != nil {
		return ""
	}
	size, _, _ := procGetFileVersionInfoSize.Call(uintptr(unsafe.Pointer(path)), 0)
	if size == 0 {
		return ""
	}
	buf := make([]byte, size)
	if ok, _, _ := procGetFileVersionInfo.Call(uintptr(unsafe.Pointer(path)), 0, size, uintptr(unsafe.Pointer(&buf[0]))); ok == 0 {
		return ""
	}
	// query liefert Zeiger und Länge eines Werts – bei Text in Zeichen, sonst in Bytes.
	query := func(key string) (unsafe.Pointer, uint32) {
		k, _ := syscall.UTF16PtrFromString(key)
		var ptr unsafe.Pointer
		var n uint32
		ok, _, _ := procVerQueryValue.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(k)), uintptr(unsafe.Pointer(&ptr)), uintptr(unsafe.Pointer(&n)))
		if ok == 0 || ptr == nil {
			return nil, 0
		}
		return ptr, n
	}
	tables := []string{"040904B0", "040704B0", "040904E4", "000004B0"}
	if ptr, n := query(`\VarFileInfo\Translation`); n >= 4 {
		t := (*[2]uint16)(ptr)
		tables = append([]string{fmt.Sprintf("%04X%04X", t[0], t[1])}, tables...)
	}
	for _, key := range []string{"ProductName", "FileDescription"} {
		for _, table := range tables {
			if ptr, n := query(`\StringFileInfo\` + table + `\` + key); n > 0 {
				if s := strings.TrimSpace(syscall.UTF16ToString(unsafe.Slice((*uint16)(ptr), n))); s != "" {
					return s
				}
			}
		}
	}
	return ""
}
