# Changelog

## 1.1.0 – 22.09.2026

- Das Programm darf im Unterordner `app` liegen, der Starter eine Ebene darüber.
- Arbeitsordner ist immer der Ordner des Starters: relative Pfade und Daten landen dort,
  in beiden Aufbauten gleich.
- Neuer Platzhalter `{app}` für den Ordner mit den Programmdateien (QElectroTech nutzt ihn).

## 1.0.0 – 21.09.2026

- Ein Starter für alle portablen Programme, gesteuert über `config_starter.json`.
- Mehrere Programme je Datei – das erste vorhandene startet.
- Platzhalter `{ordner}` und `{daten}`, der Datenordner wird angelegt.
- `vorher` und `warten`: Programme, die zuerst starten, mit Pause danach (für ecoDMS).
- Fehlt `config_starter.json`, legt der Starter sie an – mit draw.io, OrcaSlicer, Creality Print,
  PrusaSlicer, QElectroTech, Mullvad Browser und ecoDMS.
- Meldungen auf Deutsch und Englisch, Fehler in der Konfiguration mit Zeile und Zeichen.
- `tools\build.bat <Ordner>` legt je Programmordner `portable_<programm>.exe` mit Icon und Namen
  des Programms an.
