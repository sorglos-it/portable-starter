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

1. Aus `dist\` den passenden Starter in den Ordner des Programms kopieren, z. B.
   `portable_drawio.exe` neben `draw.io.exe`. Für andere Programme `starter.exe` –
   der Name ist frei.
2. Doppelklicken. Beim ersten Start legt der Starter `config_starter.json` an.
3. Fehlt das Programm dort, EXE und Parameter eintragen – der Starter bietet an, die Datei zu öffnen.

Schon eingetragen und in `dist\` fertig mit eigenem Icon: draw.io, OrcaSlicer,
Creality Print, PrusaSlicer, QElectroTech, Mullvad Browser und ecoDMS. Wer bisher
`OrcaSlicerPortableStarter`, `CrealityPrintPortableStarter` oder `PrusaSlicerPortable`
nutzt, behält seinen Ordner `profile`.

**Update:** neue EXE drüberkopieren. `config_starter.json` und die Daten bleiben.

## Zwei Aufbauten

```text
Alles in einem Ordner          Programm eine Ebene tiefer
drawio\                        drawio\
├── draw.io.exe                ├── app\                ← Programm, beim Update ersetzen
├── portable_drawio.exe        │   └── draw.io.exe
├── config_starter.json        ├── daten\              ← Einstellungen
└── daten\                     ├── lib\                ← falls nötig
                               ├── portable_drawio.exe
                               └── config_starter.json
```

Der Starter sucht die EXE neben sich, sonst im Unterordner `app`. Arbeitsordner
und Daten liegen in beiden Fällen beim Starter – ein Update tauscht nur `app\`.

**Umziehen:** Programmdateien nach `app\` verschieben; Starter, JSON und
Datenordner bleiben oben. Danach `tools\verknuepfungen.bat` doppelklicken und den
Programmordner wählen: Startmenü-Einträge, Dateitypen und Deinstallationseinträge,
die ins Leere zeigen, gehen dann auf den Starter oder den neuen Ort in `app\`.
Das Werkzeug zeigt vorher alles, sichert es unter Dokumente und fragt nach.

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
- Relative Pfade gelten ab dem Starter – dort landen auch die Daten.
- `{ordner}` – Ordner des Starters. `{daten}` – dessen Unterordner `daten`, wird angelegt.
- `{app}` – Ordner mit den Programmdateien: `app\`, sonst der Ordner des Starters.
- Dateien, die man auf den Starter zieht, bekommt das Programm mit.

Braucht ein Programm vorher ein anderes, startet `vorher` dieses zuerst;
`warten` ist die Pause danach in Sekunden (0 bis 60):

```json
{
  "exe": "ecodmsclient.exe",
  "vorher": [
    { "exe": "ecodmssinglesignon.exe", "warten": 3 }
  ]
}
```

Fehler in der Datei meldet der Starter mit Zeile und Zeichen.

## Starter mit dem Icon des Programms

Programmordner auf `tools\build.bat` ziehen, gern mehrere auf einmal:

```bat
tools\build.bat "D:\Apps\drawio" "D:\Apps\OrcaSlicer"
```

Jeder Ordner bekommt `portable_<programm>.exe` mit Icon und Namen des
Programms, das der Starter dort findet – fehlt `config_starter.json`, kommt
sie dazu. Hat die EXE kein Icon, nimmt das Werkzeug eine gleichnamige
`.ico` aus dem Programmordner, sonst das Standard-Icon. Eine Kopie landet in `dist\`.

## Bauen

Voraussetzung ist [Go](https://go.dev/dl/) ab 1.24, sonst nichts.
`tools\build.bat` ohne Angabe prüft, testet und legt `starter.exe` und
`config_starter.json` in `dist\` ab. `dist\` liegt mit im Repo – wer nur
starten will, braucht kein Go.

## Sicherheit

- Startet nur `.exe`-Dateien und nie sich selbst.
- Läuft mit normalen Rechten. Verlangt ein Programm Adminrechte, fragt Windows – wie beim Doppelklick.
- Lädt Systembibliotheken nur aus `System32`, nie aus dem Programmordner.
- Kein Netzwerk, keine Registry. Der Starter beendet sich, sobald das Programm läuft.

## Donate via PayPal

Wenn dir der Starter Arbeit spart, freue ich mich über eine Spende. Danke!

[![Donate with PayPal](https://img.shields.io/badge/Donate-PayPal-00457C?logo=paypal&logoColor=white&style=for-the-badge)](https://www.paypal.com/donate/?hosted_button_id=6CDEVZGJWTNQQ)

## Lizenz

[MIT](LICENSE). Nicht verbunden mit den Herstellern der gestarteten Programme –
deren Namen und Icons in `dist\portable_*.exe` gehören ihnen.
