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

// TestStart legt den Starter ohne config_starter.json neben ein Testprogramm,
// das wie ein Eintrag der Beispiel-Konfiguration heißt, und prüft, womit das
// Programm ankommt.
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

	for _, c := range []struct {
		name, exe string
		want      func(dir string) []string // Arbeitsordner und Parameter aus der Konfiguration
	}{
		{"draw.io", "draw.io.exe", func(dir string) []string {
			return []string{dir, "--user-data-dir=" + filepath.Join(dir, "daten"), "--disable-update"}
		}},
		{"QElectroTech im Unterordner", filepath.Join("bin", "qelectrotech.exe"), func(dir string) []string {
			return []string{filepath.Join(dir, "bin"),
				"-platform", "windows:fontengine=freetype",
				"--common-elements-dir=" + dir + "/elements/",
				"--common-tbt-dir=" + dir + "/titleblocks/",
				"--lang-dir=" + dir + "/lang/",
				"--config-dir=" + dir + "/conf/",
				"-style", "plastique"}
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			copyFile(t, filepath.Join(tools, "helper.exe"), filepath.Join(dir, c.exe))
			copyFile(t, filepath.Join(tools, "starter.exe"), filepath.Join(dir, "starter.exe"))

			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second) // ein Fehlerdialog bliebe sonst stehen
			defer cancel()
			cmd := exec.CommandContext(ctx, filepath.Join(dir, "starter.exe"), "plan.drawio", "mit Leerzeichen")
			cmd.Dir = work
			if err := cmd.Run(); err != nil {
				t.Fatalf("Starter: %v", err)
			}

			out := filepath.Join(filepath.Dir(filepath.Join(dir, c.exe)), "args.txt")
			var got []byte
			for end := time.Now().Add(10 * time.Second); len(got) == 0 && time.Now().Before(end); time.Sleep(50 * time.Millisecond) {
				got, _ = os.ReadFile(out)
			}
			want := append(c.want(dir), filepath.Join(work, "plan.drawio"), "mit Leerzeichen")
			if lines := strings.Split(string(got), "\n"); !strings.EqualFold(strings.Join(lines, "|"), strings.Join(want, "|")) {
				t.Errorf("Programm bekam\n%q\nerwartet\n%q", lines, want)
			}
			if written, _ := os.ReadFile(filepath.Join(dir, "config_starter.json")); string(written) != string(example) {
				t.Error("config_starter.json nicht aus dem Beispiel angelegt")
			}
		})
	}
}
