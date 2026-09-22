# Changelog

## 1.1.1 – 22.09.2026

- Die Versionsangaben der EXEs nennen „Sorglos Thomas Weirich“.
- `tools\build.bat` baut `dist\starter.exe` bei jedem Lauf mit – `dist\` bleibt auf einem Stand.
- `tools\verknuepfungen.bat` stellt die eigenen Einträge selbst um und überlässt dem
  Admin-Fenster nur die systemweiten. Das klappt so auch mit Netzlaufwerken und einem
  eigenen Admin-Konto.
- `tools\verknuepfungen.bat` biegt Deinstallationsprogramme und fehlende Dateien in `app\`
  nie mehr auf den Starter um.
- `tools\build.bat` stürzt bei Ordnerangaben wie `D:\Apps\.\qet` nicht mehr ab.

## 1.1.0 – 22.09.2026

- Das Programm darf im Unterordner `app` liegen, der Starter eine Ebene darüber.
- Arbeitsordner ist immer der Ordner des Starters: relative Pfade und Daten landen dort,
  in beiden Aufbauten gleich.
- Neuer Platzhalter `{app}` für den Ordner mit den Programmdateien (QElectroTech nutzt ihn).
- `tools\verknuepfungen.bat` stellt nach dem Umzug Verknüpfungen, Dateitypen und
  Deinstallationseinträge um, die ins Leere zeigen – mit Sicherung und Rückfrage.
- `dist\` liegt im Repo: `starter.exe` und fertige Starter mit Icon für draw.io,
  OrcaSlicer, Creality Print, PrusaSlicer, QElectroTech, Mullvad Browser und ecoDMS.
- `tools\build.bat <Ordner>` legt von jedem Starter eine Kopie in `dist\` ab.

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
