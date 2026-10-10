// Package heavy holds pure, non-network backend work: document parsing,
// chunking, hashing, file walking helpers (journey 10-00).
//
// Rules: NO os file IO, NO net/http, NO exec here — only []byte/string in,
// structs/JSON out. IO + jail + caching stay in backend/services; network
// stays in backend/services/ai. The bridge package types the handoff.
package heavy
