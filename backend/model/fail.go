package model

import "os"

// Fail is a classified failure. nil *Fail = success — check it first, then
// use the value. Reason is one of the constants below so callers (UI, tools)
// branch on meaning, never on error text.
type Fail struct {
	Reason string
	Detail string
}

const (
	ReasonUnsupported      = "unsupported"
	ReasonNotFound         = "not_found"
	ReasonPermissionDenied = "permission_denied"
	ReasonTooLarge         = "too_large"
	ReasonParseError       = "parse_error"
)

func (f *Fail) Error() string {
	if f == nil {
		return "<nil>"
	}
	if f.Detail != "" {
		return f.Reason + ": " + f.Detail
	}
	return f.Reason
}

// FailOf classifies a plain os error (open/stat failures) into not_found /
// permission_denied / parse_error. Anything else is a parse_error — an
// honest failure, never a fabricated document.
func FailOf(err error) *Fail {
	if os.IsNotExist(err) {
		return &Fail{ReasonNotFound, err.Error()}
	}
	if os.IsPermission(err) {
		return &Fail{ReasonPermissionDenied, err.Error()}
	}
	return &Fail{ReasonParseError, err.Error()}
}
