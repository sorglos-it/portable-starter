package classes

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
)

// Texts sind die Meldungen in der Sprache des Anwenders.
type Texts map[string]string

// LoadTexts lädt lang/<code>.json aus fsys, ersatzweise Englisch.
func LoadTexts(fsys fs.FS, code string) Texts {
	for _, c := range []string{code, "en"} {
		data, err := fs.ReadFile(fsys, "lang/"+c+".json")
		if err != nil {
			continue
		}
		var t Texts
		if json.Unmarshal(data, &t) == nil {
			return t
		}
	}
	return Texts{}
}

// Get liefert den Text zu key, ersatzweise key selbst.
func (t Texts) Get(key string) string {
	if s, ok := t[key]; ok {
		return s
	}
	return key
}

// Message macht aus err den Meldungstext für den Anwender.
func (t Texts) Message(err error) string {
	var p *Problem
	if !errors.As(err, &p) {
		return describe(err)
	}
	format, ok := t[p.Key]
	if !ok {
		return p.Error()
	}
	args := make([]any, len(p.Args))
	for i, a := range p.Args {
		if e, isErr := a.(error); isErr {
			a = describe(e)
		}
		args[i] = a
	}
	return fmt.Sprintf(format, args...)
}
