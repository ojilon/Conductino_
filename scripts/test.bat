@echo off
REM scripts\test.bat — fast local checks before you code or release.
REM Usage: scripts\test.bat
REM Does: go vet + go test ./... + frontend typecheck (if pnpm exists).
setlocal
cd /d "%~dp0.."

echo [1/3] go vet ./...
go vet ./...
if errorlevel 1 goto :fail

echo [2/3] go test ./...
go test ./...
if errorlevel 1 goto :fail

echo [3/3] frontend typecheck
where pnpm >nul 2>nul
if errorlevel 1 (
  echo   pnpm not found — skipping frontend typecheck.
  goto :ok
)
call pnpm --dir frontend exec tsc --noEmit
if errorlevel 1 goto :fail

:ok
echo ALL TESTS PASSED.
exit /b 0
:fail
echo TESTS FAILED — fix before release.
exit /b 1
