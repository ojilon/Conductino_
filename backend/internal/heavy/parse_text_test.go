package heavy

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestParseTextHeadings(t *testing.T) {
	raw := []byte("# Title\n\nBody para.\n\n## Sub\n")
	js, title, kind, reason, _ := ParseText("notes/test.md", raw)
	if reason != "" || kind != "text" || title != "test" {
		t.Fatalf("unexpected result: %q %q %q %q", js, title, kind, reason)
	}
	var wire struct {
		Blocks []struct {
			Type  string `json:"type"`
			Level int    `json:"level"`
		} `json:"blocks"`
	}
	if err := json.Unmarshal([]byte(js), &wire); err != nil {
		t.Fatal(err)
	}
	if len(wire.Blocks) != 3 || wire.Blocks[0].Type != "heading" || wire.Blocks[0].Level != 1 {
		t.Fatalf("bad blocks: %+v", wire.Blocks)
	}
}

func TestParseTextStableIDs(t *testing.T) {
	raw := []byte("Same.\n\nSame.\n")
	a, _, _, _, _ := ParseText("k", raw)
	b, _, _, _, _ := ParseText("k", raw)
	if a != b {
		t.Fatal("IDs not stable across re-parse")
	}
	if strings.Count(a, "\"id\":\"b-") != 2 {
		t.Fatalf("expected 2 ids: %s", a)
	}
}

func TestParseTextTooLarge(t *testing.T) {
	raw := make([]byte, MaxTextBytes+1)
	_, _, _, reason, _ := ParseText("k", raw)
	if reason != "too_large" {
		t.Fatalf("want too_large, got %q", reason)
	}
}

func TestParseTextEmpty(t *testing.T) {
	js, _, _, reason, _ := ParseText("k", []byte("\n\n   \n"))
	if reason != "" || !strings.Contains(js, "\"blocks\"") {
		t.Fatalf("empty should yield one empty block: %s %q", js, reason)
	}
}
