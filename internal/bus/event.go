// Package bus provides an in-process publish/subscribe signal channel.
// Payloads are plain DTO maps/structs — never sliver protobufs — so
// consumers can never be dragged into a sliver dependency.
package bus

// Event namespaces: sliver.* (raw server events), gui.* (GUI-synthesized),
// journal.*, automation.*. Integrations pick their own prefix.
type Event struct {
	Type         string
	Source       string // "grpc-stream" | "journal" | "gui" | "automation" | integration name
	ConnectionID string // "host:port" of the originating server
	Time         int64  // UnixMilli; stamped by Publish when zero
	Payload      any
}

type Handler func(Event)
