package ai

import (
	"encoding/json"
	"strings"
	"testing"

	"Conductino/backend/models"
	"Conductino/backend/skills"
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

func TestBuildPromptForNoSkillsUnchanged(t *testing.T) {
	req := models.AIRequest{Query: "What is chemiosmosis?"}
	plain, _, _ := buildPrompt(models.OpChat, req)
	wired, _, _ := (&GeminiService{}).buildPromptFor(models.OpChat, req)
	if wired != plain {
		t.Fatal("no skills loaded — prompt must be byte-identical")
	}
}

func TestBuildPromptForInjectsMatchedSkill(t *testing.T) {
	sk, err := skills.Parse([]byte("---\nskill: summarize-source\nversion: 1\nwhen: [AI_MERGE, AI_CHAT]\ntools: [read_source]\nbudgets: { maxChars: 6000 }\n---\nVerify from tool results.\n"), "test")
	if err != nil {
		t.Fatal(err)
	}
	g := &GeminiService{skills: []skills.Skill{sk}}
	prompt, _, _ := g.buildPromptFor(models.OpChat, models.AIRequest{Query: "summarize this"})
	if !strings.Contains(prompt, "## Skill: summarize-source") {
		t.Fatalf("skill excerpt missing:\n%s", prompt)
	}
	if !strings.Contains(prompt, "summarize this") {
		t.Fatal("user query lost under skill injection")
	}
	// Unmatched operation gets no injection.
	other, _, _ := g.buildPromptFor(models.OpExplain, models.AIRequest{SelectionText: "x"})
	if strings.Contains(other, "## Skill:") {
		t.Fatal("explain must not match the chat/merge skill")
	}
}

func TestBuildPromptRewriteIncludesFocusedChange(t *testing.T) {
	req := models.AIRequest{
		Query:          "Long draft sentence.",
		SummaryContent: "Current summary text.",
		FocusedChange: &models.FocusedChange{
			ID: "c1", Op: "insert", BlockID: "b9", NewContent: "Long draft sentence.",
		},
	}
	prompt, kind, _ := buildPrompt(models.OpRewrite, req)
	if kind != kindRevision {
		t.Fatalf("kind: %v", kind)
	}
	for _, want := range []string{"Long draft sentence.", "Focused pending proposal", "Current summary text."} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("rewrite prompt missing %q:\n%s", want, prompt)
		}
	}
	// No focus, no block.
	plain, _, _ := buildPrompt(models.OpRewrite, models.AIRequest{Query: "x"})
	if strings.Contains(plain, "Focused pending proposal") {
		t.Fatal("unfocused rewrite must stay bare")
	}
}

func TestEncodeChatWithProposalsCarriesTrace(t *testing.T) {
	payload := encodeChatWithProposals("done", []proposal{{Op: "insert", NewText: "x"}}, []aiToolCall{
		{Tool: "read_source", OK: true, Ms: 12},
		{Tool: "propose_summary_edit", OK: false, Ms: 3},
	}, []aiPublishedSummary{{SummaryID: "s", Path: "Summary.docx"}})
	var res aiResult
	if err := json.Unmarshal([]byte(payload), &res); err != nil {
		t.Fatal(err)
	}
	if len(res.Proposals) != 1 || len(res.ToolTrace) != 2 || len(res.Published) != 1 {
		t.Fatalf("payload: %+v", res)
	}
	if res.ToolTrace[0].Tool != "read_source" || !res.ToolTrace[0].OK || res.ToolTrace[1].OK {
		t.Fatalf("trace: %+v", res.ToolTrace)
	}
}

func TestBuildPromptChatIncludesHistory(t *testing.T) {
	req := models.AIRequest{
		Query: "What is chemiosmosis?",
		MessageHistory: []models.ChatTurn{
			{Role: "user", Content: "Hi"},
			{Role: "assistant", Content: "Hello"},
		},
	}
	prompt, kind, tokens := buildPrompt(models.OpChat, req)
	if prompt == "" || kind != kindExplanation || tokens <= 0 {
		t.Fatalf("bad: %q %v %d", prompt, kind, tokens)
	}
	if !strings.Contains(prompt, "What is chemiosmosis?") {
		t.Fatal("current query missing")
	}
	if !strings.Contains(prompt, "Hello") {
		t.Fatal("history missing")
	}
}
