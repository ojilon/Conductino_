@echo off
REM scripts\build.bat — compile app + installer to build\bin\ (local, no tag).
REM Usage: scripts\build.bat
REM Needs: Go, Node+pnpm, Wails CLI on PATH.
setlocal
cd /d "%~dp0.."

echo [1/3] frontend build (pnpm install + build)...
call pnpm --dir frontend install
if errorlevel 1 goto :fail
call pnpm --dir frontend build
if errorlevel 1 goto :fail

echo [2/3] wails build (Conductino.exe)...
wails build
if errorlevel 1 goto :fail

echo [3/3] installer build (Conductino-Installer.exe)...
if not exist "build\bin" mkdir "build\bin"
go build -o "build\bin\Conductino-Installer.exe" ./installer
if errorlevel 1 goto :fail

echo.
echo BUILD OK:
dir /b "build\bin\*.exe"
echo Next: run the installer locally, or scripts\release.bat v0.0.3 for a packaged folder.
exit /b 0
:fail
echo BUILD FAILED.
exit /b 1
