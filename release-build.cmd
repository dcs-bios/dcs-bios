@echo off

if "%MSI_VERSION%" == "" (
    echo Error: Please set the MSI_VERSION variable, e.g. "set MSI_VERSION=0.1.2.3"
    echo MSI_VERSION is the version of the MSI installer, which is used to determine if
    echo a newer version of DCS-BIOS is already installed.
    exit 1
)
if "%BUILD_VERSION%" == "" (
    echo Error: Please set the BUILD_VERSION variable, e.g. "set BUILD_VERSION=v0.1.2-alpha3"
    echo BUILD_VERSION is the version shown in the user interface next to the git commit hash.
    exit 1
)

rem this is the equivalent of BUILD_COMMIT=$(git rev-parse HEAD) in bash
FOR /F "tokens=* USEBACKQ" %%g IN (`git rev-parse HEAD`) do (SET "BUILD_COMMIT=%%g")

echo on
@echo deleting ./build directory
if exist build rd /S /Q build
@echo creating empty build directory
mkdir build

@echo building backend
cd src\hub-backend
if "%APPVEYOR%" == "True" (
    go build -ldflags "-X main.gitSha1=%BUILD_COMMIT% -X main.gitTag=%BUILD_VERSION% -H=windowsgui" -o dcs-bios-hub.exe
) else (
    go build -trimpath -ldflags "-X main.gitSha1=%BUILD_COMMIT% -X main.gitTag=%BUILD_VERSION% -H=windowsgui" -o dcs-bios-hub.exe
)

copy dcs-bios-hub.exe ..\..\build\dcs-bios-hub.exe
cd ..\..

@echo building frontend
cd src\hub-frontend
call npm install
call npm run build
cd ..\..

@echo creating installer

"%WIX%\bin\heat" dir src\dcs-lua -var var.DcsLuaSourceDir -dr DCSLuaDir -cg CMP_DcsLuaFiles -ag -g1 -sfrag -srd -out build\wix\dcs-lua.wxs
"%WIX%\bin\heat" dir src\control-reference-json -var var.ControlReferenceJsonSourceDir -dr ControlReferenceJsonDir -cg CMP_ControlReferenceJsonFiles -ag -g1 -sfrag -srd -out build\wix\control-reference-json.wxs
"%WIX%\bin\heat" dir src\hub-frontend\build -var var.FrontendAppSourceDir -dr FrontendAppDir -cg CMP_FrontendApp -gg -g1 -sfrag -srd -out build\wix\frontend-app.wxs

"%WIX%\bin\candle" -dMsiVersion=%MSI_VERSION% -out build\ -dDcsLuaSourceDir=src\dcs-lua -dControlReferenceJsonSourceDir=src\control-reference-json -dFrontendAppSourceDir=src\hub-frontend\build -arch x64 -fips -pedantic -wx -ext WixUIExtension src\installer\*.wxs build\wix\*.wxs -out build\wix\
"%WIX%\bin\light" -dMsiVersion=%MSI_VERSION% -loc src\installer\custom-text.wxl -out build\DCS-BIOS-Hub-Setup-%BUILD_VERSION%.msi build\wix\*.wixobj -ext WixUIExtension

if exist "DCS-BIOS-Hub-Setup-%BUILD_VERSION%.msi" (
    echo built version %BUILD_VERSION% (%BUILD_COMMIT%) with MSI_VERSION=%MSI_VERSION%, saved to build/DCS-BIOS-Hub-Setup-%BUILD_VERSION%.msi
    exit 0
)
exit 1


