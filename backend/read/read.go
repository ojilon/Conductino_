//implementation of the open-file loop

package read

// Fail is a classified failure. nil *Fail = success, check it first.
// Reason lets the frontend show honest messages without parsing text.
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


// Doc is one opened file. Blocks is the {"blocks":[...]} JSON string
// the React renderer parses.
type Doc struct {
	Title  string
	Kind   string // "text" for now; pdf/docx land in later cycles
	Blocks string
	Pages  int
	Root   string // absolute library root — tags which folder this came from
}