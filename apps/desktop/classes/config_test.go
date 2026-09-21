package classes

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// wantProblem prüft, dass err ein Problem mit key und args ist.
func wantProblem(t *testing.T, err error, key string, args ...any) {
	t.Helper()
	var p *Problem
	if !errors.As(err, &p) {
		t.Fatalf("Fehler %v, erwartet Problem %s", err, key)
	}
	if p.Key != key || (len(args) > 0 && !reflect.DeepEqual(p.Args, args)) {
		t.Fatalf("Problem %s %v, erwartet %s %v", p.Key, p.Args, key, args)
	}
}

func TestParseExample(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "config", "config_starter.json.example"))
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := ParseConfig(data)
	if err != nil {
		t.Fatal(err)
	}
	var exes []string
	for _, p := range cfg.Programs {
		exes = append(exes, p.Exe)
	}
	if want := []string{"draw.io.exe", "drawio.exe", "orca-slicer.exe", "CrealityPrint.exe"}; !reflect.DeepEqual(exes, want) {
		t.Errorf("Programme %v, erwartet %v", exes, want)
	}
	if want := []string{"--user-data-dir={daten}", "--disable-update"}; !reflect.DeepEqual(cfg.Programs[0].Parameters, want) {
		t.Errorf("Parameter %v, erwartet %v", cfg.Programs[0].Parameters, want)
	}
}

func TestParseAccepts(t *testing.T) {
	for name, data := range map[string]string{
		"BOM":             "\xef\xbb\xbf" + `{"programme":[{"exe":"a.exe"}]}`,
		"ohne Parameter":  `{"programme":[{"exe":"a.exe"}]}`,
		"Großschreibung":  `{"programme":[{"exe":"A.EXE","parameter":["{Daten}","{ORDNER}"]}]}`,
		"GUID und JSON":   `{"programme":[{"exe":"a.exe","parameter":["{8E0F7A12-BFB3-4FE8-B9A5-48FD50A15A9A}","{\"a\":1}"]}]}`,
		"Pfad mit Ordner": `{"programme":[{"exe":"bin/a.exe"},{"exe":"C:\\Apps\\b.exe"}]}`,
	} {
		if _, err := ParseConfig([]byte(data)); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
	cfg, err := ParseConfig([]byte(`{"programme":[{"exe":"  a.exe "}]}`))
	if err != nil || cfg.Programs[0].Exe != "a.exe" {
		t.Errorf("Leerzeichen um exe: %v %v", cfg, err)
	}
}

func TestParseRejects(t *testing.T) {
	for _, c := range []struct {
		name, data, key string
		args            []any
	}{
		{"leer", "  \n", "config.empty", nil},
		{"Komma am Ende", "{\n  \"programme\": [,]\n}", "config.syntax", []any{2, 17}},
		{"einfacher Backslash", `{"programme":[{"exe":"C:\Apps\a.exe"}]}`, "config.syntax", []any{1, 26}},
		{"Liste statt Objekt", `[{"exe":"a.exe"}]`, "config.format", nil},
		{"falscher Schlüssel", `{"programs":[]}`, "config.key", []any{"programs"}},
		{"keine Liste", `{"programme":{"exe":"a.exe"}}`, "config.list", nil},
		{"leere Liste", `{"programme":[]}`, "config.empty", nil},
		{"ohne programme", `{}`, "config.empty", nil},
		{"Eintrag kein Objekt", `{"programme":["a.exe"]}`, "program.format", []any{1}},
		{"Tippfehler", `{"programme":[{"exe":"a.exe"},{"exe":"b.exe","paramter":[]}]}`, "program.key", []any{2, "paramter"}},
		{"exe fehlt", `{"programme":[{"parameter":[]}]}`, "program.exe", []any{1}},
		{"exe keine Zeichenkette", `{"programme":[{"exe":5}]}`, "program.exe", []any{1}},
		{"keine exe", `{"programme":[{"exe":"start.bat"}]}`, "program.notexe", []any{1, "start.bat"}},
		{"Parameter als Text", `{"programme":[{"exe":"a.exe","parameter":"--x"}]}`, "program.parameter", []any{1}},
		{"Parameter als Zahl", `{"programme":[{"exe":"a.exe","parameter":[8080]}]}`, "program.parameter", []any{1}},
		{"falscher Platzhalter", `{"programme":[{"exe":"a.exe","parameter":["--dir={data}"]}]}`, "program.placeholder", []any{1, "{data}"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := ParseConfig([]byte(c.data))
			wantProblem(t, err, c.key, c.args...)
		})
	}
}

func TestLoadCreatesMissingConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), ConfigName)
	example := []byte(`{"programme":[{"exe":"a.exe"}]}`)
	cfg, err := LoadConfig(path, example)
	if err != nil || len(cfg.Programs) != 1 {
		t.Fatalf("LoadConfig: %v %v", cfg, err)
	}
	if got, _ := os.ReadFile(path); string(got) != string(example) {
		t.Errorf("angelegt %q, erwartet %q", got, example)
	}
}

func TestLoadKeepsExistingConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), ConfigName)
	own := `{"programme":[{"exe":"eigenes.exe"}]}`
	os.WriteFile(path, []byte(own), 0o644)
	cfg, err := LoadConfig(path, []byte(`{"programme":[{"exe":"a.exe"}]}`))
	if err != nil || cfg.Programs[0].Exe != "eigenes.exe" {
		t.Fatalf("LoadConfig: %v %v", cfg, err)
	}
	if got, _ := os.ReadFile(path); string(got) != own {
		t.Error("vorhandene Datei überschrieben")
	}
}

func TestLoadRejectsHugeConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), ConfigName)
	os.WriteFile(path, []byte(strings.Repeat(" ", maxConfigSize+1)), 0o644)
	_, err := LoadConfig(path, nil)
	wantProblem(t, err, "config.size")
}

func TestFind(t *testing.T) {
	dir := t.TempDir()
	touch := func(name string) string {
		path := filepath.Join(dir, name)
		os.MkdirAll(filepath.Dir(path), 0o755)
		os.WriteFile(path, nil, 0o644)
		return path
	}
	self := touch("starter.exe")
	b := touch("b.exe")
	sub := touch(filepath.Join("bin", "c.exe"))
	os.Mkdir(filepath.Join(dir, "ordner.exe"), 0o755)

	cfg := func(exes ...string) *Config {
		c := &Config{}
		for _, e := range exes {
			c.Programs = append(c.Programs, Program{Exe: e})
		}
		return c
	}
	for _, c := range []struct {
		name string
		cfg  *Config
		want string
	}{
		{"erstes vorhandenes", cfg("a.exe", "b.exe", "bin/c.exe"), b},
		{"Unterordner", cfg("bin/c.exe", "b.exe"), sub},
		{"absoluter Pfad", cfg(b), b},
		{"Ordner zählt nicht", cfg("ordner.exe", "b.exe"), b},
		{"Starter zählt nicht", cfg("starter.exe", "b.exe"), b},
	} {
		_, got, err := c.cfg.Find(dir, self)
		if err != nil || got != c.want {
			t.Errorf("%s: %q %v, erwartet %q", c.name, got, err, c.want)
		}
	}

	_, _, err := cfg("a.exe", "starter.exe").Find(dir, self)
	wantProblem(t, err, "config.notfound", "• a.exe\n• starter.exe")
}
