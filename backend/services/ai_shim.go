package services

import (
	"Conductino/backend/services/ai"
)

// AIEventSink receives one streaming unit of an AI operation.
// Alias of ai.EventSink so backend.Backend and the Wails shell keep compiling
// without importing the nested package.
type AIEventSink = ai.EventSink

// AIService is the boundary for all model access on the Go side.
type AIService = ai.Service

// NewAI builds the backend AI service (Gemini). Implementation lives in
// package ai (backend/services/ai/); this shim preserves the historical
// services.NewAI() call site used by backend.NewBackend.
func NewAI() AIService {
	return ai.New()
}
