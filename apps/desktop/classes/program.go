package classes

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// DataDir ist der Ordner neben dem Starter, für den {daten} steht.
const DataDir = "daten"

// placeholder findet {ordner}, {daten} und Tippfehler wie {data} – aber keine
// GUIDs oder JSON-Texte, die ein Programm vielleicht als Parameter braucht.
var placeholder = regexp.MustCompile(`\{(\pL+)\}`)

// Program ist ein Eintrag in config_starter.json: welche EXE mit welchen Parametern.
type Program struct {
	Exe        string
	Parameters []string
}

// check prüft den Eintrag Nummer n (ab 1).
func (p Program) check(n int) error {
	if p.Exe == "" {
		return problem("program.exe", n)
	}
	if !strings.EqualFold(filepath.Ext(p.Exe), ".exe") {
		return problem("program.notexe", n, p.Exe)
	}
	for _, arg := range p.Parameters {
		for _, m := range placeholder.FindAllStringSubmatch(arg, -1) {
			if name := strings.ToLower(m[1]); name != "ordner" && name != "daten" {
				return problem("program.placeholder", n, m[0])
			}
		}
	}
	return nil
}

// Path ist der absolute Pfad der EXE; relative Angaben gelten ab dir.
func (p Program) Path(dir string) string {
	exe := filepath.FromSlash(p.Exe)
	if filepath.IsAbs(exe) {
		return filepath.Clean(exe)
	}
	return filepath.Join(dir, exe)
}

// Arguments setzt {ordner} und {daten} ein und hängt extra an – das sind
// Dateien, die auf den Starter gezogen oder mit ihm geöffnet wurden.
// Kommt {daten} vor, legt Arguments den Datenordner an.
func (p Program) Arguments(dir string, extra []string) ([]string, error) {
	data := filepath.Join(dir, DataDir)
	args := make([]string, 0, len(p.Parameters)+len(extra))
	for _, arg := range p.Parameters {
		var mkErr error
		arg = placeholder.ReplaceAllStringFunc(arg, func(m string) string {
			switch strings.ToLower(m[1 : len(m)-1]) {
			case "ordner":
				return dir
			case "daten":
				if err := os.MkdirAll(data, 0o755); err != nil {
					mkErr = err
				}
				return data
			}
			return m
		})
		if mkErr != nil {
			return nil, problem("start.datadir", data, mkErr)
		}
		args = append(args, arg)
	}
	for _, arg := range extra {
		args = append(args, absPath(arg))
	}
	return args, nil
}

// Start startet das Programm mit seinen Parametern und extra; exe ist der
// Pfad, den Config.Find geliefert hat.
func (p Program) Start(exe, dir string, extra []string) error {
	args, err := p.Arguments(dir, extra)
	if err != nil {
		return err
	}
	if err := Launch(exe, args); err != nil {
		return problem("start.failed", p.Exe, err)
	}
	return nil
}

// absPath macht den Pfad einer vorhandenen Datei oder eines Ordners absolut:
// Das Programm läuft in seinem eigenen Ordner, dort zeigte ein relativer Pfad
// ins Leere.
func absPath(arg string) string {
	if filepath.IsAbs(arg) {
		return arg
	}
	if _, err := os.Stat(arg); err != nil {
		return arg
	}
	if abs, err := filepath.Abs(arg); err == nil {
		return abs
	}
	return arg
}
