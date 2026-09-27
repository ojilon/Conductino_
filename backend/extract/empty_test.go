package extract

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildContextPackMirrorsTS(t *testing.T) {
	wire := `{"blocks":[
		{"id":"h1","type":"heading","level":2,"segments":[{"text":"Overview"}]},
		{"id":"p1","type":"paragraph","segments":[{"text":"Before text."}]},
		{"id":"p2","type":"paragraph","segments":[{"text":"Selected passage here."}]},
		{"id":"p3","type":"paragraph","segments":[{"text":"After text."}]}
	]}`
	pack, err := BuildContextPack(wire, PackOpts{
		Title: "Doc", BlockID: "p2", SelText: "Selected passage here.",
		RangeStart: 0, RangeEnd: 8, WindowBlocks: 1, IncludeOutline: true, Budget: 6000,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"### Selection (block p2, chars 0–8)\nSelected passage here.",
		"### Document\nDoc",
		"### Before selection\nBefore text.",
		"### After selection\nAfter text.",
		"### Outline\n## Overview",
	} {
		if !strings.Contains(pack, want) {
			t.Fatalf("pack missing %q:\n%s", want, pack)
		}
	}
	// Unknown block → no window sections, outline still present.
	pack2, err := BuildContextPack(wire, PackOpts{Title: "Doc", BlockID: "nope", SelText: "x", IncludeOutline: true})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(pack2, "Before selection") || !strings.Contains(pack2, "## Overview") {
		t.Fatalf("window handling:\n%s", pack2)
	}
	// Budget truncation keeps the machine-readable marker.
	small, err := BuildContextPack(wire, PackOpts{Title: "Doc", SelText: strings.Repeat("z", 200), Budget: 100})
	if err != nil {
		t.Fatal(err)
	}
	if len(small) > 120 || !strings.Contains(small, "[…truncated…]") {
		t.Fatalf("budget: %q", small)
	}
	// Bad JSON errors.
	if _, err := BuildContextPack("{bad", PackOpts{}); err == nil {
		t.Fatal("expected JSON error")
	}
}

func TestBlocksFromMarkdownRoundTrip(t *testing.T) {
	md := "# Title\n\nFirst paragraph.\n\n- item one\n- item two\n\n--- page 2 ---\n\nLast paragraph."
	blocks, err := BlocksFromMarkdown(md, "Summary.docx")
	if err != nil {
		t.Fatal(err)
	}
	wire, err := BlocksToJSON(blocks)
	if err != nil {
		t.Fatal(err)
	}
	back, _, err := NormalizeBlocksJSON(wire)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"# Title", "First paragraph.", "- item one", "- item two", "Last paragraph."} {
		if !strings.Contains(back, want) {
			t.Fatalf("round trip missing %q:\n%s", want, back)
		}
	}
	if strings.Contains(back, "--- page") {
		t.Fatal("page markers must not survive into blocks")
	}
	// Stable IDs: same input twice → identical ids (anchors survive publish).
	again, _ := BlocksFromMarkdown(md, "Summary.docx")
	if len(again) != len(blocks) {
		t.Fatalf("block counts differ: %d vs %d", len(blocks), len(again))
	}
	for i := range blocks {
		if blocks[i].ID != again[i].ID || blocks[i].ID == "" {
			t.Fatalf("unstable id at %d: %q vs %q", i, blocks[i].ID, again[i].ID)
		}
	}
	if empty, _ := BlocksFromMarkdown("", "x"); len(empty) != 1 {
		t.Fatalf("blank md must yield one empty block, got %d", len(empty))
	}
}

func TestNormalizeBlocksJSON(t *testing.T) {
	wire := `{"blocks":[
		{"id":"h1","type":"heading","level":1,"segments":[{"text":"Chemiosmosis"}]},
		{"id":"p1","type":"paragraph","segments":[{"text":"Mitchell proposed coupling."}]},
		{"id":"pg2","type":"page","segments":[{"text":""}]},
		{"id":"p2","type":"paragraph","segments":[{"text":"Later work confirmed it."}]},
		{"id":"l1","type":"list","segments":[],"listItems":[[{"text":"a"}],[{"text":"b"}]]}
	]}`
	md, pmJSON, err := NormalizeBlocksJSON(wire)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"# Chemiosmosis", "Mitchell proposed coupling.", "--- page 2 ---", "Later work confirmed it.", "- a\n- b"} {
		if !strings.Contains(md, want) {
			t.Fatalf("md missing %q:\n%s", want, md)
		}
	}
	var pm []PageMapEntry
	if err := json.Unmarshal([]byte(pmJSON), &pm); err != nil {
		t.Fatal(err)
	}
	if len(pm) != 2 || pm[0].BlockID != "h1" || pm[1].BlockID != "p2" {
		t.Fatalf("page map: %+v", pm)
	}
	if _, _, err := NormalizeBlocksJSON("{bad"); err == nil {
		t.Fatal("expected JSON error")
	}
	if got := PlainToMarkdown("a\n\n\nb\r\n\r\n"); got != "a\n\nb" {
		t.Fatalf("plain: %q", got)
	}
}

// Brand-new 0-byte files must open as blank pages — never errors — so a
// Summary.docx created on disk (or an empty .txt/.md/.pdf) is immediately
// usable by the renderer and the summary edit loop.
func TestEmptyFilesOpenAsBlank(t *testing.T) {
	dir := t.TempDir()
	d := NewDocuments()
	cases := []struct{ name, kind string }{
		{"Summary.docx", "docx"},
		{"notes.txt", "text"},
		{"notes.md", "text"},
		{"scan.pdf", "pdf"},
	}
	for _, c := range cases {
		p := filepath.Join(dir, c.name)
		if err := os.WriteFile(p, []byte{}, 0o644); err != nil {
			t.Fatal(err)
		}
		opened, err := d.OpenFile(p, c.name)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", c.name, err)
		}
		if opened.Reason != "" {
			t.Fatalf("%s: unexpected reason %q (%s)", c.name, opened.Reason, opened.Detail)
		}
		if opened.Kind != c.kind {
			t.Fatalf("%s: kind %q, want %q", c.name, opened.Kind, c.kind)
		}
		var wrap struct {
			Blocks []struct {
				ID       string `json:"id"`
				Type     string `json:"type"`
				Segments []struct {
					Text string `json:"text"`
				} `json:"segments"`
			} `json:"blocks"`
		}
		if err := json.Unmarshal([]byte(opened.BlocksJSON), &wrap); err != nil {
			t.Fatalf("%s: bad blocks JSON: %v", c.name, err)
		}
		if len(wrap.Blocks) != 1 || wrap.Blocks[0].Type != "paragraph" {
			t.Fatalf("%s: want one empty paragraph, got %+v", c.name, wrap.Blocks)
		}
		want := "Summary"
		if c.name != "Summary.docx" {
			want = c.name[:len(c.name)-len(filepath.Ext(c.name))]
		}
		if opened.Title != want {
			t.Fatalf("%s: title %q, want %q", c.name, opened.Title, want)
		}
	}
}
