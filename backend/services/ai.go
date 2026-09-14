// Package services — Go-side services behind the Wails boundary.
//
// Every service exposes an interface + a mock implementation. The mock
// is what runs today; a real implementation is swapped in without
// touching the frontend (see docs/future-work.md §"Replacing mocks").
package services

import (
	"context"
	"time"

	"github.com/lumen/desktop/backend/models"
)

// AIEventSink receives one streaming unit of an AI operation.
type AIEventSink func(ev models.AIEvent)

// AIService is the boundary for all model access on the Go side.
//
// FUTURE INTEGRATION: implement AIService with a concrete provider
// (OpenAI / Anthropic / local Ollama). The provider selection and API
// keys belong here — never in the frontend.
type AIService interface {
	Run(ctx context.Context, req models.AIRequest, sink AIEventSink)
	ProviderName() string
	Configured() bool
}

// MockAIService simulates a provider so the desktop app is fully
// demonstrable offline. Replace with a real implementation later.
type MockAIService struct{}

func NewAI() *MockAIService { return &MockAIService{} }

func (m *MockAIService) ProviderName() string { return "Go mock provider" }
func (m *MockAIService) Configured() bool     { return false }

func (m *MockAIService) Run(ctx context.Context, req models.AIRequest, sink AIEventSink) {
	steps := []struct {
		phase int
		label string
		wait  time.Duration
	}{
		{0, "Searching", 650 * time.Millisecond},
		{1, "Finding sources", 750 * time.Millisecond},
		{2, "Comparing sources", 850 * time.Millisecond},
		{3, "Ranking results", 650 * time.Millisecond},
		{4, "Preparing useful sources", 550 * time.Millisecond},
	}
	for _, s := range steps {
		select {
		case <-ctx.Done():
			return
		case <-time.After(s.wait):
		}
		sink(models.AIEvent{Type: "phase", Phase: s.phase, Label: s.label})
	}
	sink(models.AIEvent{Type: "sources", SourceIDs: []string{"ws-1", "ws-2", "ws-3", "ws-4", "ws-5"}})
	sink(models.AIEvent{Type: "done", Payload: "mock complete"})
}
