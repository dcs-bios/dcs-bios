@echo off

REM go build -ldflags -H=windowsgui .
go build .
if ERRORLEVEL 1 goto fail
goto done

:fail
echo Compilation failed.
goto quit

:done

start dcs-bios-hub.exe
:quit