package tools

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"
	"unicode/utf8"
)

// The AI's BASIC SANDBOXED SHELL (tool name: "shell").
//
// Scope on purpose (per the app guide in backend/skills/app-guide.md):
//   - commands run with the open research folder as working directory,
//   - a hard 10-second timeout kills anything long-running,
//   - combined stdout+stderr is capped at ~8000 characters,
//   - a small deny-list rejects obviously destructive commands.
//
// This is NOT a real security boundary — it is a guard rail so the model
// can do basic things (list files, count words, peek at CSVs) without the
// user worrying about typos turning into disasters. Never run it as admin.

const (
	shellTimeout   = 10 * time.Second
	shellOutputCap = 8000 // characters returned to the model
)

// shellDenyPatterns are lowercase substrings that get rejected outright.
// Keep this list short and obvious — it complements, not replaces, care.
var shellDenyPatterns = []string{
	"rm -rf /",
	"rm -rf ~",
	"rm -rf .",
	"mkfs",
	"dd if=",
	":(){",
	"shutdown",
	"reboot",
	"init 0",
	"format c:",
	"del /f /s /q",
	"rd /s /q c:",
	"> /dev/sd",
	"chmod -r 777 /",
	"curl | sh",
	"wget | sh",
}

// RunShell executes one command in the sandbox described above. workDir is
// the directory the command starts in (the library root when one is open).
// It returns the (capped) combined output; a non-zero exit is NOT an error —
// the model should see what the command printed, including its own failure.
func RunShell(workDir, command string) (string, error) {
	command = strings.TrimSpace(command)
	if command == "" {
		return "", fmt.Errorf("shell: empty command")
	}
	lower := strings.ToLower(command)
	for _, bad := range shellDenyPatterns {
		if strings.Contains(lower, bad) {
			return "", fmt.Errorf("shell: command rejected by the sandbox deny-list (%q)", bad)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), shellTimeout)
	defer cancel()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/C", command)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
	}
	if workDir != "" {
		cmd.Dir = workDir
	}

	out, err := cmd.CombinedOutput()
	text := string(out)
	if !utf8.ValidString(text) {
		text = strings.ToValidUTF8(text, "")
	}
	truncated := false
	if utf8.RuneCountInString(text) > shellOutputCap {
		runes := []rune(text)
		text = string(runes[:shellOutputCap])
		truncated = true
	}

	var sb strings.Builder
	if ctx.Err() == context.DeadlineExceeded {
		sb.WriteString(fmt.Sprintf("[killed after %s timeout]\n", shellTimeout))
	}
	sb.WriteString(text)
	if text == "" && err != nil {
		sb.WriteString("(no output) ")
		sb.WriteString(err.Error())
	} else if err != nil {
		// Exit status text helps the model react ("command not found", …).
		sb.WriteString("\n[exit] ")
		sb.WriteString(err.Error())
	}
	if truncated {
		sb.WriteString("\n… [output truncated at 8000 chars]")
	}
	return sb.String(), nil
}
