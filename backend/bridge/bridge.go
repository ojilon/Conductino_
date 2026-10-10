// Package bridge is the narrow contract between heavy (non-network) and
// network parts of the backend (journey 10-01, 10-09).
//
// Rules:
//   - heavy/ never imports net/http; net callers never parse bytes directly.
//   - Both sides exchange these structs (versioned), never internal types.
//   - Reason strings match models.OpenFailureReason values
//     (unsupported | not_found | permission_denied | too_large | parse_error)
//     so services.ReasonOf mapping stays single-sourced in services/.
package bridge

// Version of this contract. Bump when fields change; both sides check it in tests.
const Version = 1

// Failure reasons (mirror of models.OpenFailureReason — kept as plain
// strings here so heavy/ stays dependency-free).
const (
	ReasonUnsupported       = "unsupported"
	ReasonNotFound          = "not_found"
	ReasonPermissionDenied  = "permission_denied"
	ReasonTooLarge          = "too_large"
	ReasonParseError        = "parse_error"
)

// ParseRequest describes WHAT to parse. Raw bytes travel as a separate
// argument (heavy.ParseText), never inside this struct, so the contract
// stays JSON-serializable for a future Rust sidecar (journey 11-08).
type ParseRequest struct {
	// PathToken is the workspace-relative opaque token (FileTreeNode.Path).
	PathToken string `json:"pathToken"`
	// RelKey seeds stable content-addressed block IDs.
	RelKey string `json:"relKey"`
	// KindHint is the lowercased extension without dot (txt, md, docx, pdf).
	KindHint string `json:"kindHint"`
}

// ParseResult is the outcome. Reason == "" means success.
type ParseResult struct {
	BlocksJSON string `json:"blocksJSON,omitempty"`
	Title      string `json:"title,omitempty"`
	Kind       string `json:"kind,omitempty"`
	Reason     string `json:"reason,omitempty"`
	Detail     string `json:"detail,omitempty"`
}
