// Command installer is the Conductino local installer (Windows-first).
//
// What it does, in plain terms:
//  1. Lists drives (C:\, D:\, ...) found on this machine.
//  2. Asks you to pick a drive, then an app folder on it:
//     - option 1: X:\Conductino            (drive root + fixed name)
//     - option 2: X:\<your-subfolder>\Conductino (subfolder of the drive root)
//  3. Creates the tree (or repairs it on update):
//
//     <appRoot>\                     <- e.g. D:\Conductino
//       bin\Conductino.exe
//       bin\Conductino-Installer.exe
//       Conductino_Data\              <- ALL writable app data lives here
//         config.json
//         install-info.json
//         conductino.db               (created on first app run, preserved)
//         skills\
//         cache\extract\
//         work\summaries\
//         temporary\extract_dump\
//
//  4. Copies Conductino.exe + itself into bin\.
//  5. On UPDATE (Conductino_Data already exists): overwrites binaries,
//     creates any MISSING folders, but NEVER deletes cache, db, skills,
//     summaries. That is how updates preserve user data.
//
// Usage:
//
//	Conductino-Installer.exe [--src DIR] [--app-root PATH] [--yes]
//
//	--src DIR      folder containing Conductino.exe to install (default: folder
//	               next to this installer, falling back to build\bin)
//	--app-root    skip all prompts and install here (e.g. D:\Conductino)
//	--yes         answer yes to the final confirm (for scripts)
//
// Build:
//
//	go build -o build\bin\Conductino-Installer.exe ./installer
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"Conductino/backend/version"
)

func main() {
	srcFlag := flag.String("src", "", "folder containing Conductino.exe (default: next to installer, else build\\bin)")
	rootFlag := flag.String("app-root", "", "install without prompts, e.g. D:\\Conductino")
	yesFlag := flag.Bool("yes", false, "skip final confirmation")
	flag.Parse()

	if err := run(*srcFlag, *rootFlag, *yesFlag); err != nil {
		fmt.Fprintf(os.Stderr, "installer failed: %v\n", err)
		os.Exit(1)
	}
}

func run(srcDir, appRoot string, autoYes bool) error {
	fmt.Printf("Conductino installer v%s\n\n", version.Version)

	// 1. Find source binaries.
	if srcDir == "" {
		srcDir = defaultSrcDir()
	}
	appExe := filepath.Join(srcDir, "Conductino.exe")
	if _, err := os.Stat(appExe); err != nil {
		return fmt.Errorf("Conductino.exe not found in %q (pass --src DIR). Build first: scripts\\build.bat", srcDir)
	}
	selfExe, _ := os.Executable()

	// 2. Pick target folder (interactive unless --app-root).
	var err error
	if appRoot == "" {
		appRoot, err = pickAppRoot()
		if err != nil {
			return err
		}
	}
	appRoot = strings.TrimSpace(appRoot)
	if appRoot == "" {
		return fmt.Errorf("empty install path")
	}
	if !filepath.IsAbs(appRoot) {
		return fmt.Errorf("install path must be absolute, got %q", appRoot)
	}

	dataRoot := filepath.Join(appRoot, "Conductino_Data")
	isUpdate := dirExists(dataRoot)
	if isUpdate {
		fmt.Printf("\nExisting data folder found at %s — running UPDATE (cache/db/skills preserved).\n", dataRoot)
	} else {
		fmt.Printf("\nFresh install to %s\n", appRoot)
	}
	fmt.Printf("  program: %s\n  data:    %s\n", filepath.Join(appRoot, "bin"), dataRoot)

	if !autoYes {
		fmt.Print("\nProceed? [Y/n] ")
		if !askYes(os.Stdin) {
			fmt.Println("cancelled.")
			return nil
		}
	}

	// 3. Create/verify tree. MkdirAll is idempotent: missing folders are
	// created, existing ones (with user data) are left alone. This is the
	// whole "never crash on deleted temp folder / preserve cache on update"
	// rule in one call.
	binDir := filepath.Join(appRoot, "bin")
	dirs := []string{
		binDir,
		dataRoot,
		filepath.Join(dataRoot, "skills"),
		filepath.Join(dataRoot, "cache", "extract"),
		filepath.Join(dataRoot, "work", "summaries"),
		filepath.Join(dataRoot, "temporary", "extract_dump"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("create %q: %w", d, err)
		}
	}

	// 4. Copy binaries (overwrite = update).
	if err := copyFile(appExe, filepath.Join(binDir, "Conductino.exe")); err != nil {
		return err
	}
	if selfExe != "" {
		if _, err := os.Stat(selfExe); err == nil {
			_ = copyFile(selfExe, filepath.Join(binDir, "Conductino-Installer.exe"))
		}
	}

	// 5. config.json only if missing (never overwrite user settings).
	cfgPath := filepath.Join(dataRoot, "config.json")
	if !fileExists(cfgPath) {
		cfg := map[string]any{"dataRoot": dataRoot, "createdBy": "installer v" + version.Version}
		raw, _ := json.MarshalIndent(cfg, "", "  ")
		_ = os.WriteFile(cfgPath, raw, 0o644)
	}

	// 6. install-info.json always rewritten (audit of last install/update).
	info := map[string]any{
		"version":     version.Version,
		"installedAt": time.Now().Format(time.RFC3339),
		"appRoot":     appRoot,
		"dataRoot":    dataRoot,
		"update":      isUpdate,
	}
	raw, _ := json.MarshalIndent(info, "", "  ")
	_ = os.WriteFile(filepath.Join(dataRoot, "install-info.json"), raw, 0o644)

	fmt.Printf("\nDone. Run %s\n", filepath.Join(binDir, "Conductino.exe"))
	fmt.Printf("Data lives in %s (set CONDUCTINO_DATA to it, or the app finds it via portable ./data).\n", dataRoot)
	fmt.Println("To point a dev run at it: set CONDUCTINO_DATA=" + dataRoot)
	return nil
}

// listDrives checks A:\..Z:\ for existence. Simple and dependency-free;
// network drives that are disconnected just don't appear.
func listDrives() []string {
	var out []string
	for c := 'A'; c <= 'Z'; c++ {
		p := string([]rune{c}) + `:\`
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			out = append(out, p)
		}
	}
	return out
}

func pickAppRoot() (string, error) {
	drives := listDrives()
	if len(drives) == 0 {
		return "", fmt.Errorf("no drives found")
	}
	fmt.Println("Available drives:")
	for i, d := range drives {
		fmt.Printf("  %d) %s\n", i+1, d)
	}
	fmt.Print("Choose drive [1]: ")
	idx := askInt(os.Stdin, 1, len(drives)) - 1
	drive := drives[idx]

	fmt.Printf("\nDrive %s selected.\n", drive)
	fmt.Println("Where should the app live?")
	fmt.Printf("  1) %s (drive root, recommended)\n", filepath.Join(drive, "Conductino"))
	fmt.Println("  2) A subfolder of the drive root (e.g. Apps\\Conductino)")
	fmt.Print("Choose [1]: ")
	mode := askInt(os.Stdin, 1, 2)
	if mode == 1 {
		return filepath.Join(drive, "Conductino"), nil
	}
	fmt.Print("Subfolder path under " + drive + " [Apps\\Conductino]: ")
	sub := askLine(os.Stdin)
	if strings.TrimSpace(sub) == "" {
		sub = filepath.Join("Apps", "Conductino")
	}
	// Jail the subfolder to the chosen drive: reject absolute outsiders and .. escapes.
	clean := filepath.Clean(sub)
	if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("subfolder must stay under %s, got %q", drive, sub)
	}
	return filepath.Join(drive, clean), nil
}

func defaultSrcDir() string {
	if exe, err := os.Executable(); err == nil {
		if d := filepath.Dir(exe); d != "" {
			if _, err := os.Stat(filepath.Join(d, "Conductino.exe")); err == nil {
				return d
			}
		}
	}
	// Repo-layout fallback: <repo>/build/bin (installer run via `go run ./installer`).
	if cwd, err := os.Getwd(); err == nil {
		cand := filepath.Join(cwd, "build", "bin")
		if _, err := os.Stat(filepath.Join(cand, "Conductino.exe")); err == nil {
			return cand
		}
		// cwd might be installer/ itself.
		cand2 := filepath.Join(filepath.Dir(cwd), "build", "bin")
		if _, err := os.Stat(filepath.Join(cand2, "Conductino.exe")); err == nil {
			return cand2
		}
	}
	return "build\\bin"
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func dirExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

func askLine(r *os.File) string {
	sc := bufio.NewScanner(r)
	if sc.Scan() {
		return strings.TrimSpace(sc.Text())
	}
	return ""
}

func askInt(r *os.File, lo, hi int) int {
	for {
		line := askLine(r)
		if line == "" {
			return lo
		}
		var n int
		if _, err := fmt.Sscanf(line, "%d", &n); err == nil && n >= lo && n <= hi {
			return n
		}
		fmt.Printf("Enter a number %d-%d: ", lo, hi)
	}
}

func askYes(r *os.File) bool {
	line := strings.ToLower(askLine(r))
	return line == "" || line == "y" || line == "yes"
}
