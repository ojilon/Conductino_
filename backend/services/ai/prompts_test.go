package ai

import (
	"strings"
	"testing"

	"Conductino/backend/models"
)

func TestWithContextIncludesCustomAndPack(t *testing.T) {
	req := models.AIRequest{
		CustomPrompt: "Focus on the mechanism",
		ContextPack:  "### Selection\nATP synthase turns",
	}
	out := withContext("BASE", req)
	if !strings.Contains(out, "BASE") {
		t.Fatal("missing base")
	}
	if !strings.Contains(out, "User instruction") || !strings.Contains(out, "Focus on the mechanism") {
		t.Fatal("missing custom prompt")
	}
	if !strings.Contains(out, "Document context") || !strings.Contains(out, "ATP synthase") {
		t.Fatal("missing context pack")
	}
}

func TestBuildPromptExplainUsesSelectionAndContext(t *testing.T) {
	req := models.AIRequest{
		SelectionText: "chemiosmosis",
		ContextPack:   "### Outline\n# Methods",
		CustomPrompt:  "Be brief",
	}
	prompt, kind, tokens := buildPrompt(models.OpExplain, req)
	if prompt == "" || kind != kindExplanation || tokens <= 0 {
		t.Fatalf("bad prompt result: %q %v %d", prompt, kind, tokens)
	}
	if !strings.Contains(prompt, "chemiosmosis") {
		t.Fatal("selection missing")
	}
	if !strings.Contains(prompt, "Be brief") {
		t.Fatal("custom prompt missing")
	}
	if !strings.Contains(prompt, "Methods") {
		t.Fatal("context pack missing")
	}
}

func TestBuildPromptUnknownOp(t *testing.T) {
	p, k, n := buildPrompt(models.AIOperation("AI_NOPE"), models.AIRequest{})
	if p != "" || k != "" || n != 0 {
		t.Fatalf("expected empty, got %q %v %d", p, k, n)
	}
}
