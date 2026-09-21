package classes

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"testing/fstest"
)

func loadLang(t *testing.T, code string) Texts {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "lang", code+".json"))
	if err != nil {
		t.Fatal(err)
	}
	var texts Texts
	if err := json.Unmarshal(data, &texts); err != nil {
		t.Fatalf("%s.json: %v", code, err)
	}
	return texts
}

// TestTextsComplete prüft, dass jede Meldung aus dem Code in beiden Sprachen
// da ist, dort gleich viele Platzhalter hat und keine übrig ist.
func TestTextsComplete(t *testing.T) {
	used := map[string]bool{}
	files, _ := filepath.Glob("*.go")
	call := regexp.MustCompile(`(?:problem|Get)\("([a-z.]+)"`)
	for _, f := range append(files, filepath.Join("..", "main.go")) {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range call.FindAllStringSubmatch(string(src), -1) {
			used[m[1]] = true
		}
	}
	langs := map[string]Texts{"de": loadLang(t, "de"), "en": loadLang(t, "en")}
	verbs := regexp.MustCompile(`%[a-z]`)
	for key := range used {
		for code, texts := range langs {
			if _, ok := texts[key]; !ok {
				t.Errorf("%s.json: %q fehlt", code, key)
			}
		}
		if de, en := verbs.FindAllString(langs["de"][key], -1), verbs.FindAllString(langs["en"][key], -1); strings.Join(de, "") != strings.Join(en, "") {
			t.Errorf("%q: Platzhalter %v (de) und %v (en)", key, de, en)
		}
	}
	for code, texts := range langs {
		for key := range texts {
			if !used[key] {
				t.Errorf("%s.json: %q wird nirgends benutzt", code, key)
			}
		}
	}
}

func TestLoadTextsFallsBackToEnglish(t *testing.T) {
	fsys := fstest.MapFS{"lang/en.json": {Data: []byte(`{"title":"Starter"}`)}}
	if got := LoadTexts(fsys, "fr").Get("title"); got != "Starter" {
		t.Errorf("%q, erwartet Starter", got)
	}
	if got := LoadTexts(fstest.MapFS{}, "de").Get("title"); got != "title" {
		t.Errorf("%q, erwartet den Schlüssel", got)
	}
}

func TestMessage(t *testing.T) {
	texts := Texts{"config.syntax": "Zeile %d, Zeichen %d", "config.read": "nicht lesbar: %v"}
	if got := texts.Message(problem("config.syntax", 3, 7)); got != "Zeile 3, Zeichen 7" {
		t.Errorf("%q", got)
	}
	_, err := os.Open(filepath.Join(t.TempDir(), "fehlt.json"))
	got := texts.Message(problem("config.read", err))
	if !strings.HasPrefix(got, "nicht lesbar: ") || strings.Contains(got, "open ") || strings.Contains(got, "fehlt.json") {
		t.Errorf("Windows-Meldung erwartet, ohne Go-Vorsatz: %q", got)
	}
}
