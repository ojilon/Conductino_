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

// NewAI builds the backend AI service (Gemini) with folder-scoped tools.
// fs and docs may be nil (tools degrade gracefully; chat still works).
func NewAI(fs *Filesystem, docs *Documents) AIService {
	if fs == nil && docs == nil {
		return ai.New()
	}
	return ai.NewWithTools(fs, docs)
}
