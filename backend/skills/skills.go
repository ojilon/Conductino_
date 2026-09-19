package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Skill is one instruction file: machine routing in frontmatter, prose steps
// in the body. Skills teach the harness HOW to use tools (verify, retry,
// confirm, report) — tools remain the only code that acts. Skills never
// contain keys, paths, or file content.
//
// Format (one file per skill, see summarize-source.md):
//
//	---
//	skill: summarize-source
//	version: 1
//	when: ["user asks to summarize", "AI_MERGE"]
//	tools: [list_workspace, read_source, read_summary, propose_summary_edit]
//	budgets: { maxToolCalls: 6, maxChars: 6000 }
//	---
//	# Summarize a source into the living summary
//	1. ...
type Skill struct {
	Name    string
	Version int
	When    []string
	Tools   []string
	Budgets map[string]int
	Body    string
	Source  string // file path the skill was loaded from ("" when parsed directly)
}

// Parse reads one skill file: YAML-ish frontmatter between --- lines plus a
// Markdown body. Only the shapes above are supported (scalars, [lists],
// {k: v} int maps) — full YAML is deliberately out of scope; keep skill
// files to this shape so the stdlib parser stays total.
func Parse(data []byte, source string) (Skill, error) {
	var s Skill
	s.Source = source
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	lines := strings.Split(text, "\n")
	if len(lines) < 3 || strings.TrimSpace(lines[0]) != "---" {
		return s, fmt.Errorf("skills: %s missing frontmatter ---", display(source))
	}
	closeIdx := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			closeIdx = i
			break
		}
	}
	if closeIdx < 0 {
		return s, fmt.Errorf("skills: %s unclosed frontmatter", display(source))
	}
	head := strings.Join(lines[1:closeIdx], "\n")
	s.Body = strings.TrimSpace(strings.Join(lines[closeIdx+1:], "\n"))
	fm := map[string]string{}
	for _, line := range strings.Split(head, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			return s, fmt.Errorf("skills: %s bad frontmatter line %q", display(source), line)
		}
		fm[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	s.Name = strings.Trim(fm["skill"], `"'`)
	if s.Name == "" {
		return s, fmt.Errorf("skills: %s frontmatter needs skill: <name>", display(source))
	}
	if fm["version"] != "" {
		v, err := strconv.Atoi(strings.Trim(fm["version"], `"'`))
		if err != nil {
			return s, fmt.Errorf("skills: %s bad version %q", display(source), fm["version"])
		}
		s.Version = v
	}
	var err error
	if s.When, err = parseList(fm["when"]); err != nil {
		return s, fmt.Errorf("skills: %s bad when: %v", display(source), err)
	}
	if s.Tools, err = parseList(fm["tools"]); err != nil {
		return s, fmt.Errorf("skills: %s bad tools: %v", display(source), err)
	}
	if s.Budgets, err = parseMap(fm["budgets"]); err != nil {
		return s, fmt.Errorf("skills: %s bad budgets: %v", display(source), err)
	}
	if strings.TrimSpace(s.Body) == "" {
		return s, fmt.Errorf("skills: %s empty body", display(source))
	}
	return s, nil
}

func display(source string) string {
	if source == "" {
		return "<input>"
	}
	return source
}

// parseList reads [a, b, "c"] (brackets optional for a single value).
func parseList(v string) ([]string, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil, nil
	}
	if strings.HasPrefix(v, "[") {
		if !strings.HasSuffix(v, "]") {
			return nil, fmt.Errorf("unclosed list %q", v)
		}
		v = v[1 : len(v)-1]
	}
	var out []string
	for _, part := range strings.Split(v, ",") {
		part = strings.Trim(strings.TrimSpace(part), `"'`)
		if part != "" {
			out = append(out, part)
		}
	}
	return out, nil
}

// parseMap reads { maxToolCalls: 6, maxChars: 6000 } (braces required).
func parseMap(v string) (map[string]int, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil, nil
	}
	if !strings.HasPrefix(v, "{") || !strings.HasSuffix(v, "}") {
		return nil, fmt.Errorf("budgets must be {k: v} %q", v)
	}
	out := map[string]int{}
	for _, part := range strings.Split(v[1:len(v)-1], ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		k, val, ok := strings.Cut(part, ":")
		if !ok {
			return nil, fmt.Errorf("bad budget entry %q", part)
		}
		n, err := strconv.Atoi(strings.TrimSpace(val))
		if err != nil {
			return nil, fmt.Errorf("bad budget value %q", part)
		}
		out[strings.TrimSpace(k)] = n
	}
	return out, nil
}

// LoadDir parses every *.md skill in dir (bundled, workspace, or user
// layer — the caller decides the directory). One bad file fails the load
// so a typo'd skill can never silently vanish.
func LoadDir(dir string) ([]Skill, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []Skill
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".md") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		s, err := Parse(raw, path)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}

// Match returns skills whose `when` mentions the operation or intent
// (case-insensitive). Operation matches by substring either way; intent
// matches when a when-phrase appears inside it ("add to summary" fires on
// "please add this to summary") — so routing works for skills no operation
// names. The prompt assembler caps the result (at most 2 excerpts, ~1500
// chars total — see Excerpt).
func Match(skills []Skill, operation, intent string) []Skill {
	op := strings.ToLower(operation)
	in := strings.ToLower(intent)
	var out []Skill
	for _, s := range skills {
		for _, w := range s.When {
			wl := strings.ToLower(w)
			if (op != "" && strings.Contains(wl, op)) ||
				(in != "" && (strings.Contains(wl, in) || strings.Contains(in, wl))) {
				out = append(out, s)
				break
			}
		}
	}
	return out
}

// Excerpt renders one skill for prompt injection, capped at maxChars.
// Long skills get skipped by every model — keep files to one screen.
func Excerpt(s Skill, maxChars int) string {
	var b strings.Builder
	b.WriteString("## Skill: " + s.Name + "\n")
	body := strings.TrimSpace(s.Body)
	if maxChars > 0 && len(body) > maxChars {
		body = body[:maxChars] + "\n…[truncated]"
	}
	b.WriteString(body)
	return b.String()
}
