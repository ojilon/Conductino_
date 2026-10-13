@echo off
REM scripts\push_release.bat — tag + GitHub pre-release (needs gh CLI + clean git).
REM Usage: scripts\push_release.bat v0.0.3
REM Does: git tag -a, git push tag, gh release create --prerelease with notes + zip.
setlocal
cd /d "%~dp0.."

if "%~1"=="" (
  echo Usage: scripts\push_release.bat v0.0.3
  exit /b 1
)
set "ARG=%~1"
if "%ARG:~0,1%"=="v" set "ARG=%ARG:~1%"
set "TAG=v%ARG%"
set "ZIP=installer\dist\Conductino-%TAG%-windows-amd64.zip"
set "NOTES=docs\release-prep\release-notes\%TAG%.md"

where gh >nul 2>nul
if errorlevel 1 (
  echo gh CLI not found. Install from https://cli.github.com/ and run: gh auth login
  exit /b 1
)

REM tag if missing
git rev-parse "%TAG%" >nul 2>nul
if errorlevel 1 (
  echo Creating tag %TAG%...
  git tag -a "%TAG%" -m "%TAG%"
  if errorlevel 1 goto :fail
) else (
  echo Tag %TAG% already exists locally.
)

echo Pushing tag %TAG%...
git push origin "%TAG%"
if errorlevel 1 goto :fail

REM notes body: per-tag file wins, else CHANGELOG hint, else stub
set "BODYFILE=%NOTES%"
if not exist "%BODYFILE%" set "BODYFILE=CHANGELOG.md"

echo Creating pre-release %TAG% on GitHub...
if exist "%ZIP%" (
  gh release create "%TAG%" "%ZIP%" --title "%TAG%" --notes-file "%BODYFILE%" --prerelease
) else (
  echo WARNING: %ZIP% missing — run scripts\release.bat %TAG% first. Uploading notes only.
  gh release create "%TAG%" --title "%TAG%" --notes-file "%BODYFILE%" --prerelease
)
if errorlevel 1 goto :fail

echo PUSHED: pre-release %TAG% live. Promote to stable manually on GitHub when ready.
exit /b 0
:fail
echo PUSH FAILED.
exit /b 1
