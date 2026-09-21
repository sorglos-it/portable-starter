@echo off
rem Baut den Portable Starter.
rem   tools\build.bat                        dist\starter.exe mit Standard-Icon
rem   tools\build.bat "D:\Apps\drawio" ...   je Ordner portable_<programm>.exe mit dem
rem                                          Icon des Programms - Ordner einfach auf
rem                                          diese Datei ziehen
setlocal
set "STARTER_CWD=%CD%"
where go >nul 2>&1 || (echo Go fehlt: https://go.dev/dl/ & pause & exit /b 1)

pushd "%~dp0..\apps\desktop"
go vet ./... && go test -count=1 ./...
set "RESULT=%ERRORLEVEL%"
popd
if not "%RESULT%"=="0" (echo Tests fehlgeschlagen. & pause & exit /b %RESULT%)

pushd "%~dp0"
go run . %*
set "RESULT=%ERRORLEVEL%"
popd
pause
exit /b %RESULT%
