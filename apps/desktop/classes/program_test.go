package classes

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestPath(t *testing.T) {
	dir := `C:\Apps\drawio`
	for exe, want := range map[string]string{
		"draw.io.exe":       `C:\Apps\drawio\draw.io.exe`,
		"bin/app.exe":       `C:\Apps\drawio\bin\app.exe`,
		`bin\app.exe`:       `C:\Apps\drawio\bin\app.exe`,
		`D:\Tools\x.exe`:    `D:\Tools\x.exe`,
		"D:/Tools/../x.exe": `D:\x.exe`,
	} {
		if got := (Program{Exe: exe}).Path(dir); got != want {
			t.Errorf("%q: %q, erwartet %q", exe, got, want)
		}
	}
}

func TestArguments(t *testing.T) {
	dir := t.TempDir()
	app := filepath.Join(dir, AppDir)
	p := Program{Parameters: []string{"--user-data-dir={daten}", "--root={Ordner}", "--lang={APP}/lang/", "mit Leerzeichen", "{8E0F7A12-BFB3}"}}
	got, err := p.Arguments(dir, app, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"--user-data-dir=" + filepath.Join(dir, DataDir), "--root=" + dir, "--lang=" + app + "/lang/", "mit Leerzeichen", "{8E0F7A12-BFB3}"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%q, erwartet %q", got, want)
	}
	if info, err := os.Stat(filepath.Join(dir, DataDir)); err != nil || !info.IsDir() {
		t.Error("Datenordner nicht angelegt")
	}
}

func TestArgumentsWithoutDataKeepsFolderClean(t *testing.T) {
	dir := t.TempDir()
	if _, err := (Program{Parameters: []string{"--datadir", "profile"}}).Arguments(dir, dir, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, DataDir)); !os.IsNotExist(err) {
		t.Error("Datenordner ohne {daten} angelegt")
	}
}

func TestArgumentsFindsExistingFoldersBesideStarter(t *testing.T) {
	dir := t.TempDir()
	os.Mkdir(filepath.Join(dir, "profile"), 0o755)
	got, err := (Program{Parameters: []string{"--datadir", "profile", "-style", "plastique"}}).Arguments(dir, t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"--datadir", filepath.Join(dir, "profile"), "-style", "plastique"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%q, erwartet %q", got, want)
	}
}

func TestArgumentsMakesExtraFilesAbsolute(t *testing.T) {
	work := t.TempDir()
	// t.Chdir findet auf einem Netzlaufwerk nicht zurück, deshalb von Hand
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(work); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(old) })
	os.WriteFile("plan.drawio", nil, 0o644)
	got, err := (Program{}).Arguments(t.TempDir(), t.TempDir(), []string{"plan.drawio", ".", "--neu", `C:\x.txt`})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{filepath.Join(work, "plan.drawio"), work, "--neu", `C:\x.txt`}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%q, erwartet %q", got, want)
	}
}

func TestArgumentsReportsDataDirError(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, DataDir), nil, 0o644) // Datei statt Ordner
	_, err := (Program{Parameters: []string{"{daten}"}}).Arguments(dir, dir, nil)
	wantProblem(t, err, "start.datadir")
}
