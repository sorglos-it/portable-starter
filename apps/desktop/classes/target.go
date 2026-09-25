package classes

import (
	"path/filepath"
	"time"
)

// AppDir ist der Unterordner mit den Programmdateien, wenn der Starter eine
// Ebene über dem Programm liegt.
const AppDir = "app"

// Target ist das Programm, das Config.Find gefunden hat, mit seinen Ordnern.
type Target struct {
	Program
	Dir string // Ordner des Starters: {ordner}, Daten
	App string // Ordner der Programmdateien: {app}, ab hier gelten „exe“ und „vorher“
	Exe string // Pfad der EXE
}

// roots sind die Ordner, in denen der Starter die Programme sucht: sein
// eigener und dessen Unterordner app.
func roots(dir string) []string {
	return []string{dir, filepath.Join(dir, AppDir)}
}

// Start startet erst die „vorher“-Programme, jeweils mit ihrer Pause, dann das
// Programm mit seinen Parametern und extra.
func (t Target) Start(extra []string) error {
	for _, b := range t.Before {
		if err := t.launch(b, b.Path(t.App), nil); err != nil {
			return err
		}
		time.Sleep(b.Wait)
	}
	return t.launch(t.Program, t.Exe, extra)
}

func (t Target) launch(p Program, exe string, extra []string) error {
	args, err := p.Arguments(t.Dir, t.App, extra)
	if err != nil {
		return err
	}
	// Arbeitsordner ist der Ordner der EXE, wie beim Doppelklick: Manche
	// Programme (ecoDMS) laden Plugins relativ dazu.
	if err := Launch(exe, args, filepath.Dir(exe)); err != nil {
		return problem("start.failed", p.Exe, err)
	}
	return nil
}
