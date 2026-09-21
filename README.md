# Portable Starter

[![Platform](https://img.shields.io/badge/platform-Windows-0078D6?logo=windows&logoColor=white)](#)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Donate with PayPal](https://img.shields.io/badge/Donate-PayPal-00457C?logo=paypal&logoColor=white)](https://www.paypal.com/donate/?hosted_button_id=6CDEVZGJWTNQQ)

Ein Starter für alle portablen Windows-Programme. Er startet das Programm so,
dass dessen Einstellungen im Programmordner bleiben statt im Benutzerprofil.
Welches Programm mit welchen Parametern, steht in `config_starter.json`.

| Ordner | Zweck | Sprache | Start | Build |
| --- | --- | --- | --- | --- |
| `apps/desktop` | Starter `starter.exe` | Go | `dist\starter.exe` | `tools\build.bat` |

## Start in 3 Schritten

1. `starter.exe` in den Ordner des Programms kopieren, z. B. neben `draw.io.exe`.
2. `starter.exe` doppelklicken. Beim ersten Start legt er `config_starter.json` an.
3. Fehlt das Programm dort, EXE und Parameter eintragen – der Starter bietet an, die Datei zu öffnen.

Schon eingetragen: draw.io, OrcaSlicer und Creality Print. Wer bisher
`OrcaSlicerPortableStarter` oder `CrealityPrintPortableStarter` nutzt, behält
seinen Ordner `profile`.

**Update:** neue `starter.exe` drüberkopieren. `config_starter.json` und die Daten bleiben.

## config_starter.json

```json
{
  "programme": [
    {
      "exe": "draw.io.exe",
      "parameter": ["--user-data-dir={daten}", "--disable-update"]
    },
    {
      "exe": "orca-slicer.exe",
      "parameter": ["--datadir", "profile"]
    }
  ]
}
```

- Der Starter nimmt das **erste** Programm der Liste, das in seinem Ordner liegt.
  Eine Datei passt so für viele Programme.
- `exe` – Dateiname, auch mit Unterordner: `bin/programm.exe`.
- `parameter` – ein Eintrag je Parameter. Leerzeichen brauchen keine Anführungszeichen.
- `{ordner}` – Ordner des Starters. `{daten}` – dessen Unterordner `daten`, wird angelegt.
- Relative Pfade gelten ab dem Ordner des Programms.
- Dateien, die man auf den Starter zieht, bekommt das Programm mit.

Fehler in der Datei meldet der Starter mit Zeile und Zeichen.

## Eigenes Icon

```bat
tools\build.bat "D:\Apps\drawio\draw.io.exe"
```

baut den Starter mit dem Icon dieses Programms – oder einer `.ico`-Datei.
Ohne Angabe gibt es das Standard-Icon.

## Bauen

Voraussetzung ist [Go](https://go.dev/dl/) ab 1.24, sonst nichts.
`tools\build.bat` prüft, testet und legt `starter.exe` und
`config_starter.json` in `dist\` ab.

## Sicherheit

- Startet nur `.exe`-Dateien und nie sich selbst.
- Läuft mit normalen Rechten. Verlangt ein Programm Adminrechte, fragt Windows – wie beim Doppelklick.
- Lädt Systembibliotheken nur aus `System32`, nie aus dem Programmordner.
- Kein Netzwerk, keine Registry. Der Starter beendet sich, sobald das Programm läuft.

## Donate via PayPal

Wenn dir der Starter Arbeit spart, freue ich mich über eine Spende. Danke!

[![Donate with PayPal](https://img.shields.io/badge/Donate-PayPal-00457C?logo=paypal&logoColor=white&style=for-the-badge)](https://www.paypal.com/donate/?hosted_button_id=6CDEVZGJWTNQQ)

## Lizenz

[MIT](LICENSE). Nicht verbunden mit den Herstellern der gestarteten Programme.
