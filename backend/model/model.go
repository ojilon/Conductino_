// Package model is the single home for every JSON shape crossing a
// boundary: Wails wire types (frontend domain twins), storage records,
// failure reasons. One package, no aliases — if two places need the same
// shape, it lives here.
//
// Import rule: model imports stdlib ONLY (JSON tags are plain strings, so
// even encoding/json is unneeded here). It must never import backend/*:
// everyone imports model, model imports no one.
package model
