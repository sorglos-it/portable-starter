package classes

import (
	"errors"
	"fmt"
	"strings"
)

// Problem ist ein Fehler, den der Anwender selbst beheben kann. Key wählt den
// Meldungstext aus lang/*.json, Args füllen dessen Platzhalter.
type Problem struct {
	Key  string
	Args []any
}

func problem(key string, args ...any) *Problem {
	return &Problem{Key: key, Args: args}
}

func (p *Problem) Error() string {
	return fmt.Sprintf("%s %v", p.Key, p.Args)
}

// InConfig meldet, ob sich err durch Bearbeiten von config_starter.json beheben lässt.
func InConfig(err error) bool {
	var p *Problem
	return errors.As(err, &p) && (strings.HasPrefix(p.Key, "config.") || strings.HasPrefix(p.Key, "program."))
}
