@echo off
rem Nach dem Umzug von Programmen in den Unterordner app: stellt Verknuepfungen,
rem Dateitypen und Deinstallationseintraege auf die portable_*.exe um.
rem Doppelklicken und den Ordner mit den Programmen waehlen - oder den Ordner
rem (z. B. D:\App\PortableApps) auf diese Datei ziehen.
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0verknuepfungen.ps1" %*
