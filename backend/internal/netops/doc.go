// Package netops documents what STAYS on the network side of the split
// (journey 10-02): AI HTTP clients, failover, streaming events, update
// checks, prompt assembly with live context, tool-loop orchestration,
// usage metering. No parsing code lives here; it calls bridge types +
// heavy results, never heavy internals directly.
package netops
