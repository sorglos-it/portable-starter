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
		n := i + 1
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
			return nil, problem("program.format", n)
		}
		var p Program
		for _, key := range slices.Sorted(maps.Keys(fields)) {
			switch key {
			case "exe":
				if json.Unmarshal(fields[key], &p.Exe) != nil {
					return nil, problem("program.exe", n)
				}
			case "parameter":
				if json.Unmarshal(fields[key], &p.Parameters) != nil {
					return nil, problem("program.parameter", n)
				}
			default:
				return nil, problem("program.key", n, key)
			}
		}
		p.Exe = strings.TrimSpace(p.Exe)
		if err := p.check(n); err != nil {
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

// Find liefert das erste Programm, dessen EXE vorhanden ist, samt Pfad. Der
// Starter selbst zählt nie mit – er würde sich sonst endlos neu starten.
func (c *Config) Find(dir, self string) (Program, string, error) {
	selfInfo, selfErr := os.Stat(self)
	names := make([]string, 0, len(c.Programs))
	for _, p := range c.Programs {
		path := p.Path(dir)
		info, err := os.Stat(path)
		if err == nil && info.Mode().IsRegular() && (selfErr != nil || !os.SameFile(info, selfInfo)) {
			return p, path, nil
		}
		names = append(names, p.Exe)
	}
	return Program{}, "", problem("config.notfound", "• "+strings.Join(names, "\n• "))
}
