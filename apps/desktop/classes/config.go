package classes

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"maps"
	"os"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

// ConfigName ist der Name der Konfiguration neben dem Starter.
const ConfigName = "config_starter.json"

// maxConfigSize begrenzt, wie viel der Starter einliest.
const maxConfigSize = 1 << 20

// Config ist der Inhalt von config_starter.json.
type Config struct {
	Programs []Program
}

// LoadConfig liest path. Fehlt die Datei, legt LoadConfig sie mit example an;
// geht das nicht (schreibgeschützter Ordner), gilt example trotzdem.
func LoadConfig(path string, example []byte) (*Config, error) {
	data, err := readConfig(path)
	if errors.Is(err, fs.ErrNotExist) {
		writeNew(path, example)
		return ParseConfig(example)
	}
	if err != nil {
		return nil, err
	}
	return ParseConfig(data)
}

func readConfig(path string) ([]byte, error) {
	f, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	if err != nil {
		return nil, problem("config.read", err)
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxConfigSize+1))
	if err != nil {
		return nil, problem("config.read", err)
	}
	if len(data) > maxConfigSize {
		return nil, problem("config.size")
	}
	return data, nil
}

// writeNew legt path an, überschreibt aber nie eine vorhandene Datei und
// lässt keine halbe zurück.
func writeNew(path string, data []byte) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return
	}
	_, err = f.Write(data)
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		os.Remove(path)
	}
}

// ParseConfig prüft den Inhalt von config_starter.json und liefert die
// Programme in der eingetragenen Reihenfolge.
func ParseConfig(data []byte) (*Config, error) {
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf")) // BOM, schreibt der Windows-Editor gern
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, problem("config.empty")
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal(data, &top); err != nil {
		var syntax *json.SyntaxError
		if errors.As(err, &syntax) {
			line, col := position(data, syntax.Offset)
			return nil, problem("config.syntax", line, col)
		}
		return nil, problem("config.format")
	}
	for _, key := range slices.Sorted(maps.Keys(top)) {
		if key != "programme" {
			return nil, problem("config.key", key)
		}
	}
	var entries []json.RawMessage
	if raw, ok := top["programme"]; ok {
		if err := json.Unmarshal(raw, &entries); err != nil {
			return nil, problem("config.list")
		}
	}

	cfg := &Config{}
	for i, raw := range entries {
		p, err := parseProgram(raw, strconv.Itoa(i+1), false)
		if err != nil {
			return nil, err
		}
		cfg.Programs = append(cfg.Programs, p)
	}
	if len(cfg.Programs) == 0 {
		return nil, problem("config.empty")
	}
	return cfg, nil
}

// position rechnet einen Byte-Versatz in Zeile und Zeichen um, beide ab 1.
func position(data []byte, offset int64) (line, col int) {
	before := data[:min(max(offset, 0), int64(len(data)))]
	line = bytes.Count(before, []byte{'\n'}) + 1
	col = utf8.RuneCount(before[bytes.LastIndexByte(before, '\n')+1:])
	return line, max(col, 1)
}

// Find liefert das erste Programm der Liste, dessen EXE neben dem Starter
// (dir) oder in dessen Unterordner app liegt, und prüft, dass seine
// „vorher“-Programme dort auch sind. Der Starter selbst zählt nie mit – er
// würde sich sonst endlos neu starten.
func (c *Config) Find(dir, self string) (Target, error) {
	selfInfo, selfErr := os.Stat(self)
	isSelf := func(info fs.FileInfo) bool { return selfErr == nil && os.SameFile(info, selfInfo) }
	names := make([]string, 0, len(c.Programs))
	for i, p := range c.Programs {
		for _, app := range roots(dir) {
			exe := p.Path(app)
			info, err := os.Stat(exe)
			if err != nil || !info.Mode().IsRegular() || isSelf(info) {
				continue
			}
			for j, b := range p.Before {
				where := beforeWhere(strconv.Itoa(i+1), j+1)
				info, err := os.Stat(b.Path(app))
				if err != nil || !info.Mode().IsRegular() {
					return Target{}, problem("program.missing", where, b.Exe)
				}
				if isSelf(info) {
					return Target{}, problem("program.self", where)
				}
			}
			return Target{Program: p, Dir: dir, App: app, Exe: exe}, nil
		}
		names = append(names, p.Exe)
	}
	return Target{}, problem("config.notfound", "• "+strings.Join(names, "\n• "))
}
