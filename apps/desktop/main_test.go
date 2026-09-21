package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// helper schreibt in <eigener Name>.txt neben sich, wann und womit es
// gestartet wurde: Startzeit in ms, Arbeitsordner, dann je Zeile ein Argument.
const helper = `package main

import (
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	start := strconv.FormatInt(time.Now().UnixMilli(), 10)
	exe, _ := os.Executable()
	wd, _ := os.Getwd()
	lines := append([]string{start, wd}, os.Args[1:]...)
	os.WriteFile(strings.TrimSuffix(exe, ".exe")+".txt", []byte(strings.Join(lines, "\n")), 0o644)
}
`

func build(t *testing.T, out string, args ...string) {
	t.Helper()
	cmd := exec.Command("go", append([]string{"build", "-ldflags=-H=windowsgui", "-o", out}, args...)...)
	if msg, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build %s: %v\n%s", out, err, msg)
	}
}

func copyFile(t *testing.T, from, to string) {
	t.Helper()
	data, err := os.ReadFile(from)
	if err == nil {
		err = os.MkdirAll(filepath.Dir(to), 0o755)
	}
	if err == nil {
		err = os.WriteFile(to, data, 0o755)
	}
	if err != nil {
		t.Fatal(err)
	}
}

// started wartet auf den Bericht des Testprogramms exe und liefert Startzeit
// und die übrigen Zeilen (Arbeitsordner, Argumente).
func started(t *testing.T, exe string) (time.Time, []string) {
	t.Helper()
	var got []byte
	for end := time.Now().Add(15 * time.Second); len(got) == 0 && time.Now().Before(end); time.Sleep(50 * time.Millisecond) {
		got, _ = os.ReadFile(strings.TrimSuffix(exe, ".exe") + ".txt")
	}
	lines := strings.Split(string(got), "\n")
	ms, err := strconv.ParseInt(lines[0], 10, 64)
	if err != nil {
		t.Fatalf("%s wurde nicht gestartet", filepath.Base(exe))
	}
	return time.UnixMilli(ms), lines[1:]
}

func sameLines(got, want []string) bool {
	return strings.EqualFold(strings.Join(got, "|"), strings.Join(want, "|"))
}

// TestStart legt den Starter ohne config_starter.json neben Testprogramme,
// die wie Einträge der Beispiel-Konfiguration heißen, und prüft, womit und in
// welcher Reihenfolge sie ankommen.
func TestStart(t *testing.T) {
	if testing.Short() {
		t.Skip("baut zwei Programme")
	}
	tools := t.TempDir()
	src := filepath.Join(tools, "helper.go")
	if err := os.WriteFile(src, []byte(helper), 0o644); err != nil {
		t.Fatal(err)
	}
	build(t, filepath.Join(tools, "helper.exe"), src)
	build(t, filepath.Join(tools, "starter.exe"), ".")

	work := t.TempDir() // Datei relativ zum Aufrufer, wie bei „Öffnen mit“ aus der Konsole
	if err := os.WriteFile(filepath.Join(work, "plan.drawio"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	extra := []string{filepath.Join(work, "plan.drawio"), "mit Leerzeichen"}

	// setup legt die Testprogramme exes und den Starter in einen neuen Ordner,
	// startet ihn wie ein Doppelklick mit zwei Dateien und liefert den Ordner.
	setup := func(t *testing.T, exes ...string) string {
		dir := t.TempDir()
		for _, exe := range exes {
			copyFile(t, filepath.Join(tools, "helper.exe"), filepath.Join(dir, exe))
		}
		copyFile(t, filepath.Join(tools, "starter.exe"), filepath.Join(dir, "starter.exe"))
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second) // ein Fehlerdialog bliebe sonst stehen
		defer cancel()
		cmd := exec.CommandContext(ctx, filepath.Join(dir, "starter.exe"), "plan.drawio", "mit Leerzeichen")
		cmd.Dir = work
		if err := cmd.Run(); err != nil {
			t.Fatalf("Starter: %v", err)
		}
		if written, _ := os.ReadFile(filepath.Join(dir, "config_starter.json")); string(written) != string(example) {
			t.Error("config_starter.json nicht aus dem Beispiel angelegt")
		}
		return dir
	}

	t.Run("draw.io", func(t *testing.T) {
		dir := setup(t, "draw.io.exe")
		_, got := started(t, filepath.Join(dir, "draw.io.exe"))
		want := append([]string{dir, "--user-data-dir=" + filepath.Join(dir, "daten"), "--disable-update"}, extra...)
		if !sameLines(got, want) {
			t.Errorf("draw.io bekam\n%q\nerwartet\n%q", got, want)
		}
	})

	t.Run("QElectroTech im Unterordner", func(t *testing.T) {
		exe := filepath.Join("bin", "qelectrotech.exe")
		dir := setup(t, exe)
		_, got := started(t, filepath.Join(dir, exe))
		want := append([]string{filepath.Join(dir, "bin"),
			"-platform", "windows:fontengine=freetype",
			"--common-elements-dir=" + dir + "/elements/",
			"--common-tbt-dir=" + dir + "/titleblocks/",
			"--lang-dir=" + dir + "/lang/",
			"--config-dir=" + dir + "/conf/",
			"-style", "plastique"}, extra...)
		if !sameLines(got, want) {
			t.Errorf("QElectroTech bekam\n%q\nerwartet\n%q", got, want)
		}
	})

	t.Run("ecoDMS mit vorher und warten", func(t *testing.T) {
		dir := setup(t, "ecodmssinglesignon.exe", "ecodmsclient.exe")
		signOn, got := started(t, filepath.Join(dir, "ecodmssinglesignon.exe"))
		if !sameLines(got, []string{dir}) {
			t.Errorf("Anmeldung bekam %q, erwartet nur den Arbeitsordner", got)
		}
		client, got := started(t, filepath.Join(dir, "ecodmsclient.exe"))
		if want := append([]string{dir}, extra...); !sameLines(got, want) {
			t.Errorf("Client bekam\n%q\nerwartet\n%q", got, want)
		}
		if pause := client.Sub(signOn); pause < 2900*time.Millisecond {
			t.Errorf("Client %v nach der Anmeldung gestartet, erwartet 3 s", pause)
		}
	})
}
