@echo off
rem Baut dist\starter.exe und dist\config_starter.json.
rem Mit dem Icon eines Programms:  tools\build.bat "D:\Apps\drawio\draw.io.exe"
rem (oder eine .ico-Datei). Ohne Angabe: Standard-Icon.
setlocal
cd /d "%~dp0.."

where go >nul 2>&1 || (echo Go fehlt: https://go.dev/dl/ & exit /b 1)

set "ICON=%~1"
if not defined ICON set "ICON=apps\desktop\assets\icon.ico"
set /p VERSION=<apps\desktop\VERSION

go run tools\winres.go -icon "%ICON%" -version %VERSION% -out apps\desktop\rsrc_windows_amd64.syso || exit /b 1

if not exist dist mkdir dist
pushd apps\desktop
go vet ./... && go test -count=1 ./... && go build -trimpath -ldflags "-s -w -H windowsgui" -o ..\..\dist\starter.exe .
set "RESULT=%ERRORLEVEL%"
popd
if not "%RESULT%"=="0" (echo Build fehlgeschlagen. & exit /b %RESULT%)

copy /y apps\desktop\config\config_starter.json.example dist\config_starter.json >nul
echo Fertig: dist\starter.exe und dist\config_starter.json, Version %VERSION%
