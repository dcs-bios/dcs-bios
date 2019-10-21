@echo off

del dcs-bios-hub.exe 2>NUL
cd hub-backend
REM go build -ldflags -H=windowsgui .
go build -trimpath -o ..\dcs-bios-hub.exe .
cd ..

dcs-bios-hub.exe
