// Package docpipe is the "mirror pipeline" between real files on disk and
// the TypeScript frontend. It is the heart of how Conductino displays and
// edits documents.
//
// THE BIG PICTURE (read this first)
//
// A source file (report.docx, paper.pdf, notes.txt) is written in a format
// the browser cannot edit directly (DOCX is a zip of XML, PDF is a binary
// page language). Instead of fighting those formats in the frontend, we
// mirror every document into TWO small JSON files the frontend owns:
//
//	source file (docx / pdf / txt / md)
//	    │  backend/extract        — reads the real format (ZIP+OOXML, pdf lib, …)
//	    ▼
//	wire blocks JSON              — {blocks:[{id,type,segments},…]} (existing shape)
//	    │  docpipe.Build          — the "process" step: frees the content from
//	    ▼                           pdf/docx rules and records how the source looked
//	content.json  +  meta.json    — the MIRROR pair, stored under
//	                                <app-data>/cache/extract/mirrors/<key>/
//	    │
//	    ▼
//	frontend MirrorCanvas (TS)    — renders custom pages/fonts from meta.json,
//	                                edits content blocks freely (no XML anywhere)
//	    │  on Save
//	    ▼
//	docpipe.WriteBack             — verifies which blocks actually changed
//	                                (per-block hash from meta.json) and writes
//	                                the source file back (.docx / .txt / .md)
//
// WHY TWO FILES?
//
//   - content.json is FORMAT-FREE: plain blocks of text with light style
//     hints. The frontend can render it on any custom canvas, highlight it,
//     feed it to the AI, or edit it — never touching OOXML or PDF syntax.
//   - meta.json is the GUIDE: it remembers how the content sat in the
//     original file (kind, page size, default font, margins, which page
//     each block was on, per-block text hashes, source stamp). The frontend
//     uses it to *mirror* the look of the source (page sizes, fonts) in its
//     own way, and WriteBack uses the hashes to verify only the parts that
//     changed before saving.
//
// The AI reads the same pair (see DescribeForAI): the metadata tells it how
// the user is currently seeing the file — which page holds which content —
// so answers can reference the user's view instead of raw bytes.
package docpipe
