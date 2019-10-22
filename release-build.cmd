@echo off

FOR /F "tokens=* USEBACKQ" %%g IN (`git describe --tags`) do (SET "BUILD_VERSION=%%g")
FOR /F "tokens=* USEBACKQ" %%g IN (`git rev-parse HEAD`) do (SET "BUILD_COMMIT=%%g")

echo deleting ./build directory
rd /S /Q build
echo creating empty build directory
mkdir build
mkdir build\apps

echo building backend
cd src\hub-backend
go build -trimpath -ldflags "-X main.gitSha1=%BUILD_COMMIT% -X main.gitTag=%BUILD_VERSION% -H=windowsgui" -o  ..\..\build\dcs-bios-hub.exe

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
go-msi make --path ../src/installer/wix.json --src ../src/installer/ --arch amd64 --msi DCS-BIOS-Hub-Setup-%BUILD_VERSION%.msi --version %BUILD_VERSION% --license ../DCS-BIOS-License.txt
if exist "DCS-BIOS-Hub-Setup-%BUILD_VERSION%.msi" echo built version %BUILD_VERSION% (%BUILD_COMMIT%), saved to build/DCS-BIOS-Hub-Setup-%BUILD_VERSION%.msi
cd ..

:end