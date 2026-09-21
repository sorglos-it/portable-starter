// Portable Starter startet das erste Programm aus config_starter.json, das im
// eigenen Ordner liegt, mit den eingetragenen Parametern – so bleiben dessen
// Einstellungen im Programmordner statt im Benutzerprofil.
package main

import (
	"embed"
	"os"
	"path/filepath"

	"starter/classes"
)

//go:embed config/config_starter.json.example
var example []byte

//go:embed lang/*.json
var lang embed.FS

func main() {
	classes.HardenDLLSearch()
	config, err := run(os.Args[1:])
	if err == nil {
		return
	}
	texts := classes.LoadTexts(lang, classes.UserLanguage())
	title, message := texts.Get("title"), texts.Message(err)
	if !classes.InConfig(err) {
		classes.ShowError(title, message)
	} else if classes.Ask(title, message+"\n\n"+texts.Get("edit")) {
		if err := classes.OpenInEditor(config); err != nil {
			classes.ShowError(title, texts.Message(err))
		}
	}
	os.Exit(1)
}

// run sucht das Programm und startet es; config ist der Pfad der Konfiguration.
func run(extra []string) (config string, err error) {
	self, err := os.Executable()
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(self)
	config = filepath.Join(dir, classes.ConfigName)
	cfg, err := classes.LoadConfig(config, example)
	if err != nil {
		return config, err
	}
	program, exe, err := cfg.Find(dir, self)
	if err != nil {
		return config, err
	}
	return config, program.Start(exe, dir, extra)
}
