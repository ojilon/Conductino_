package ai

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// Native tool_calls must round-trip through the internal tag form,
// including quotes inside proposal text.
func TestToolCallToTagRoundTrip(t *testing.T) {
	tc := oaiToolCall{}
	tc.Function.Name = "propose_summary_edit"
	tc.Function.Arguments = `{"op":"modify","target":"b-1","text":"say \"hi\" now"}`
	tag := toolCallToTag(tc)
	calls := ParseToolCalls("answer\n" + tag + "\ndone")
	if len(calls) != 1 {
		t.Fatalf("want 1 call, got %d in %q", len(calls), tag)
	}
	if calls[0].Args["text"] != `say "hi" now` {
		t.Fatalf("text=%q", calls[0].Args["text"])
	}
	if calls[0].Args["op"] != "modify" || calls[0].Args["target"] != "b-1" {
		t.Fatalf("args=%v", calls[0].Args)
	}
	// Unknown names survive translation; Dispatch rejects them downstream.
	tc2 := oaiToolCall{}
	tc2.Function.Name = "bogus_tool"
	tc2.Function.Arguments = `{}`
	if got := toolCallToTag(tc2); !strings.Contains(got, "bogus_tool") {
		t.Fatalf("tag=%q", got)
	}
}

func TestToolsUnsupported(t *testing.T) {
	if !toolsUnsupported(errors.New("unsupported parameter: tool_choice")) {
		t.Fatal("should detect tool_choice rejection")
	}
	if toolsUnsupported(errors.New("429 rate limit (groq): quota")) {
		t.Fatal("quota is a model-chain matter, not a tools matter")
	}
	if toolsUnsupported(nil) {
		t.Fatal("nil must be false")
	}
}
func TestToolSchemasMirrorRegistry(t *testing.T) {
	schemas := oaiToolSchemas()
	if len(schemas) != 5 {
		t.Fatalf("want 5 schemas, got %d", len(schemas))
	}
	seen := map[string]bool{}
	for _, s := range schemas {
		seen[s.Function.Name] = true
	}
	for _, want := range []string{ToolListWorkspace, ToolReadSource, ToolReadSummary, ToolProposeSummaryEdit, ToolSearchWorkspace} {
		if !seen[want] {
			t.Fatalf("schema missing for %s", want)
		}
	}
}

func TestToolSchemasValidAgainstMetaschemaShape(t *testing.T) {
	// Groq rejects empty "required":[] — parameterless tools must omit it.
	raw, err := json.Marshal(oaiToolSchemas())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), `"required":[]`) {
		t.Fatalf("empty required array present: %s", raw)
	}
	// A metaschema rejection must trigger the no-tools retry, not kill the turn.
	if !toolsUnsupported(errors.New(`invalid JSON schema for tool list_workspace: not valid against metaschema`)) {
		t.Fatal("should detect metaschema rejection")
	}
}
