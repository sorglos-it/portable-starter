package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// helper schreibt in args.txt neben sich, womit es gestartet wurde:
// Arbeitsordner, dann je Zeile ein Argument.
const helper = `package main

import (
	"os"
	"path/filepath"
	"strings"
)

func main() {
	exe, _ := os.Executable()
	wd, _ := os.Getwd()
	lines := append([]string{wd}, os.Args[1:]...)
	os.WriteFile(filepath.Join(filepath.Dir(exe), "args.txt"), []byte(strings.Join(lines, "\n")), 0o644)
}
`

func build(t *testing.T, out string, args ...string) {
	t.Helper()
	cmd := exec.Command("go", append([]string{"build", "-ldflags=-H=windowsgui", "-o", out}, args...)...)
	if msg, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build %s: %v\n%s", out, err, msg)
	}
}

// TestStart baut den Starter und ein Testprogramm namens draw.io.exe, startet
// den Starter ohne config_starter.json und prüft, womit das Programm ankommt.
func TestStart(t *testing.T) {
	if testing.Short() {
		t.Skip("baut zwei Programme")
	}
	dir := t.TempDir()
	src := filepath.Join(t.TempDir(), "helper.go")
	if err := os.WriteFile(src, []byte(helper), 0o644); err != nil {
		t.Fatal(err)
	}
	build(t, filepath.Join(dir, "draw.io.exe"), src)
	build(t, filepath.Join(dir, "starter.exe"), ".")

	work := t.TempDir() // Datei relativ zum Aufrufer, wie bei „Öffnen mit“ aus der Konsole
	if err := os.WriteFile(filepath.Join(work, "plan.drawio"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second) // ein Fehlerdialog bliebe sonst stehen
	defer cancel()
	cmd := exec.CommandContext(ctx, filepath.Join(dir, "starter.exe"), "plan.drawio", "mit Leerzeichen")
	cmd.Dir = work
	if err := cmd.Run(); err != nil {
		t.Fatalf("Starter: %v", err)
	}

	var got []byte
	for end := time.Now().Add(10 * time.Second); len(got) == 0 && time.Now().Before(end); time.Sleep(50 * time.Millisecond) {
		got, _ = os.ReadFile(filepath.Join(dir, "args.txt"))
	}
	want := []string{
		dir,
		"--user-data-dir=" + filepath.Join(dir, "daten"),
		"--disable-update",
		filepath.Join(work, "plan.drawio"),
		"mit Leerzeichen",
	}
	if lines := strings.Split(string(got), "\n"); !strings.EqualFold(strings.Join(lines, "|"), strings.Join(want, "|")) {
		t.Errorf("Programm bekam\n%q\nerwartet\n%q", lines, want)
	}
	if info, err := os.Stat(filepath.Join(dir, "daten")); err != nil || !info.IsDir() {
		t.Error("Datenordner fehlt")
	}
	if written, _ := os.ReadFile(filepath.Join(dir, "config_starter.json")); string(written) != string(example) {
		t.Error("config_starter.json nicht aus dem Beispiel angelegt")
	}
}
