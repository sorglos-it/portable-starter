package classes

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
)

// DataDir ist der Ordner neben dem Starter, für den {daten} steht.
const DataDir = "daten"

// maxWait begrenzt „warten“, damit ein Tippfehler den Starter nicht ewig aufhält.
const maxWait = 60 * time.Second

// placeholder findet {ordner}, {daten} und Tippfehler wie {data} – aber keine
// GUIDs oder JSON-Texte, die ein Programm vielleicht als Parameter braucht.
var placeholder = regexp.MustCompile(`\{(\pL+)\}`)

// Program ist ein Eintrag in config_starter.json: welche EXE mit welchen
// Parametern und welche Programme vorher starten.
type Program struct {
	Exe        string
	Parameters []string
	Before     []Program     // „vorher“: startet der Starter zuerst, in dieser Reihenfolge
	Wait       time.Duration // „warten“: Pause nach dem Start eines „vorher“-Programms
}

// parseProgram liest einen Eintrag; where nennt ihn in Meldungen („3“ oder
// „3 (vorher 1)“). Ein „vorher“-Eintrag kennt „warten“ statt „vorher“.
func parseProgram(raw json.RawMessage, where string, before bool) (Program, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
		return Program{}, problem("program.format", where)
	}
	allowed := "exe, parameter, vorher"
	if before {
		allowed = "exe, parameter, warten"
	}
	var p Program
	for _, key := range slices.Sorted(maps.Keys(fields)) {
		value := fields[key]
		switch {
		case key == "exe":
			if json.Unmarshal(value, &p.Exe) != nil {
				return Program{}, problem("program.exe", where)
			}
		case key == "parameter":
			if json.Unmarshal(value, &p.Parameters) != nil {
				return Program{}, problem("program.parameter", where)
			}
		case key == "vorher" && !before:
			var list []json.RawMessage
			if json.Unmarshal(value, &list) != nil {
				return Program{}, problem("program.before", where)
			}
			for i, item := range list {
				b, err := parseProgram(item, beforeWhere(where, i+1), true)
				if err != nil {
					return Program{}, err
				}
				p.Before = append(p.Before, b)
			}
		case key == "warten" && before:
			var seconds float64
			if json.Unmarshal(value, &seconds) != nil || seconds < 0 || seconds > maxWait.Seconds() {
				return Program{}, problem("program.wait", where)
			}
			p.Wait = time.Duration(seconds * float64(time.Second))
		default:
			return Program{}, problem("program.key", where, key, allowed)
		}
	}
	p.Exe = strings.TrimSpace(p.Exe)
	return p, p.check(where)
}

// beforeWhere nennt den n-ten „vorher“-Eintrag von where.
func beforeWhere(where string, n int) string {
	return fmt.Sprintf("%s (vorher %d)", where, n)
}

// check prüft EXE und Platzhalter des Eintrags where.
func (p Program) check(where string) error {
	if p.Exe == "" {
		return problem("program.exe", where)
	}
	if !strings.EqualFold(filepath.Ext(p.Exe), ".exe") {
		return problem("program.notexe", where, p.Exe)
	}
	for _, arg := range p.Parameters {
		for _, m := range placeholder.FindAllStringSubmatch(arg, -1) {
			if name := strings.ToLower(m[1]); name != "ordner" && name != "daten" {
				return problem("program.placeholder", where, m[0])
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

// Start startet erst die „vorher“-Programme, jeweils mit ihrer Pause, dann das
// Programm selbst mit seinen Parametern und extra. exe ist der Pfad, den
// Config.Find geliefert hat.
func (p Program) Start(exe, dir string, extra []string) error {
	for _, b := range p.Before {
		if err := b.launch(b.Path(dir), dir, nil); err != nil {
			return err
		}
		time.Sleep(b.Wait)
	}
	return p.launch(exe, dir, extra)
}

func (p Program) launch(exe, dir string, extra []string) error {
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
