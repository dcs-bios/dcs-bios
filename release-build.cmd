@echo off

echo deleting ./build directory
rd /S /Q build
echo creating empty build directory
mkdir build
mkdir build\apps

echo building backend
cd src\hub-backend
go build -trimpath -ldflags -H=windowsgui -o  ..\..\build\dcs-bios-hub.exe
cd ..\..

echo building frontend
cd src\hub-frontend
call npm run build
cd ..\..

echo copying files to build directory
xcopy /E src\hub-frontend\build build\apps\hubconfig\
xcopy /E src\control-reference-json build\control-reference-json\
xcopy /E src\dcs-lua build\dcs-lua\
xcopy src\hub-backend\dcsbios-channel-logo.ico build\

echo creating installer
cd build
go-msi make --path ../src/installer/wix.json --src ../src/installer/ --arch amd64 --msi setup.msi --version 0.8.0 --license ../DCS-BIOS-License.txt
cd ..
