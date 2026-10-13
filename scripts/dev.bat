@echo off
REM scripts\dev.bat — your everyday debug launcher. Just run this, nothing else to type.
REM Usage: scripts\dev.bat
REM Does:
REM   1. creates .dev-data\temporary\extract_dump\ if missing (your code recreates it anyway)
REM   2. sets CONDUCTINO_DUMP_EXTRACT=1 for THIS window only (dumps each opened file to JSON)
REM   3. runs "wails dev" (blocks until you close the app window / Ctrl+C)
REM   4. on exit, wipes temporary\extract_dump\ contents (undoes the temp stuff, keeps cache/db/skills)
REM
REM Rule for all future code: never hardcode ".dev-data" or "Conductino_Data".
REM Always resolve via apppaths.DataRoot() / ExtractDumpDir() and MkdirAll on demand.
REM Then this bat gives you .dev-data behavior, and the compiled exe gives you
REM Conductino_Data behavior — same code, both work.
setlocal
cd /d "%~dp0.."

REM --- 1. setup temp dir (idempotent: missing recreated, existing left alone) ---
if not exist ".dev-data" mkdir ".dev-data" >nul 2>nul
if not exist ".dev-data\temporary\extract_dump" mkdir ".dev-data\temporary\extract_dump" >nul 2>nul
echo [dev] temp dir ready: .dev-data\temporary\extract_dump\
echo [dev] dumps always on: every opened file lands there as JSON, then the UI reads it back.
set "CONDUCTINO_DUMP_EXTRACT=1"

REM --- 2. run the app (blocks here until you close it) ---
call wails dev %*
set "EXITCODE=%ERRORLEVEL%"

REM --- 3. cleanup: undo temp dumps, preserve everything else ---
echo.
echo [dev] app closed (exit %EXITCODE%). Cleaning temporary dumps...
if exist ".dev-data\temporary\extract_dump" (
  REM delete dumped JSONs but keep the folder so next run needs no setup
  del /q ".dev-data\temporary\extract_dump\*.json" >nul 2>nul
  echo [dev] wiped .dev-data\temporary\extract_dump\*.json (cache, db, skills, work/ kept).
) else (
  echo [dev] nothing to clean.
)

endlocal
exit /b %EXITCODE%
