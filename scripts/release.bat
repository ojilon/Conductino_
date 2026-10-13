@echo off
REM scripts\release.bat — LOCAL release pack (no git tag, no GitHub).
REM Usage: scripts\release.bat v0.0.3   (or 0.0.3)
REM Does:
REM   1. normalize version, bump backend\version\version.go + wails.json
REM   2. ensure docs\release-prep\release-notes\vX.Y.Z.md exists (from TEMPLATE)
REM   3. call scripts\build.bat
REM   4. copy exes into installer\dist\Conductino-vX.Y.Z-windows-amd64\ + zip it
REM Test it by running the installer from that folder.
setlocal EnableDelayedExpansion
cd /d "%~dp0.."

if "%~1"=="" (
  echo Usage: scripts\release.bat v0.0.3
  exit /b 1
)
set "ARG=%~1"
REM strip leading v if present
if "%ARG:~0,1%"=="v" set "ARG=%ARG:~1%"
set "VER=%ARG%"
set "TAG=v%ARG%"

echo Release %TAG% (local only)...

REM --- 1. bump version.go ---
powershell -NoProfile -Command "$p='backend/version/version.go'; (Get-Content $p) -replace 'const Version = \".*\"', 'const Version = \"%VER%\"' | Set-Content $p"
if errorlevel 1 goto :fail

REM --- 2. bump wails.json productVersion ---
powershell -NoProfile -Command "$j=Get-Content wails.json -Raw | ConvertFrom-Json; $j.info.productVersion='%VER%'; ($j | ConvertTo-Json -Depth 6) | Set-Content wails.json"
if errorlevel 1 goto :fail

REM --- 3. release notes stub ---
if not exist "docs\release-prep\release-notes\%TAG%.md" (
  echo Creating docs\release-prep\release-notes\%TAG%.md from TEMPLATE — EDIT IT before push.
  copy "docs\release-prep\release-notes\TEMPLATE.md" "docs\release-prep\release-notes\%TAG%.md" >nul
)

REM --- 4. build ---
call scripts\build.bat
if errorlevel 1 goto :fail

REM --- 5. package ---
set "OUT=installer\dist\Conductino-%TAG%-windows-amd64"
if exist "%OUT%" rmdir /s /q "%OUT%"
mkdir "%OUT%\bin" >nul 2>nul
copy /y "build\bin\Conductino.exe" "%OUT%\bin\" >nul
copy /y "build\bin\Conductino-Installer.exe" "%OUT%\bin\" >nul
(
  echo Conductino %TAG% — local release
  echo.
  echo 1. Run bin\Conductino-Installer.exe, pick a drive + folder.
  echo 2. It creates Conductino_Data\ ^(skills, cache\extract, work\summaries, temporary\extract_dump^).
  echo 3. Updating? Re-run installer — data is preserved.
) > "%OUT%\README.txt"

set "ZIP=installer\dist\Conductino-%TAG%-windows-amd64.zip"
if exist "%ZIP%" del /q "%ZIP%"
powershell -NoProfile -Command "Compress-Archive -Path '%OUT%\*' -DestinationPath '%ZIP%'"
if errorlevel 1 goto :fail

echo.
echo LOCAL RELEASE OK: %OUT%  +  %ZIP%
echo Test: %OUT%\bin\Conductino-Installer.exe
exit /b 0
:fail
echo RELEASE FAILED.
exit /b 1
