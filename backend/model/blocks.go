package model

import "encoding/json"

// Segment is one styled text run. Mirrors the TS Segment.
type Segment struct {
	Text   string `json:"text"`
	Em     bool   `json:"em,omitempty"`
	Strong bool   `json:"strong,omitempty"`
}

// Block is one render unit: paragraph, heading (Level 1-3), list, or page
// break. Mirrors the TS DocumentBlock. List text lives in ListItems;
// Segments stays an (empty) array so `.map()` never sees null.
type Block struct {
	ID        string     `json:"id"`
	Type      string     `json:"type"`
	Level     int        `json:"level,omitempty"`
	Segments  []Segment  `json:"segments"`
	ListItems [][]Segment `json:"listItems,omitempty"`
}

// Text joins a block's plain text (lists join rows with newlines).
func (b Block) Text() string {
	if b.Type == "list" && len(b.ListItems) > 0 {
		rows := make([]string, 0, len(b.ListItems))
		for _, item := range b.ListItems {
			s := ""
			for _, g := range item {
				s += g.Text
			}
			rows = append(rows, s)
		}
		out := ""
		for i, r := range rows {
			if i > 0 {
				out += "\n"
			}
			out += r
		}
		return out
	}
	out := ""
	for _, s := range b.Segments {
		out += s.Text
	}
	return out
}

// MarshalBlocks encodes blocks to the {"blocks":[...]} wire string with
// encoding/json — never string-concatenated — so arbitrary file content
// cannot break the wire format.
func MarshalBlocks(blocks []Block) string {
	if len(blocks) == 0 {
		blocks = []Block{{ID: "empty", Type: "paragraph", Segments: []Segment{{Text: ""}}}}
	}
	raw, err := json.Marshal(map[string]any{"blocks": blocks})
	if err != nil {
		return `{"blocks":[]}`
	}
	return string(raw)
}

// ParseBlocks decodes the wire string back to blocks.
func ParseBlocks(blocksJSON string) ([]Block, error) {
	var env struct {
		Blocks []Block `json:"blocks"`
	}
	if err := json.Unmarshal([]byte(blocksJSON), &env); err != nil {
		return nil, err
	}
	return env.Blocks, nil
}
