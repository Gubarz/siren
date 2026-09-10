// Package events defines bus payloads shared by producers and triggers
// without coupling the automation packages to the services that emit them.
package events

type TaskResult struct {
	Verb       string `json:"verb"`
	TargetID   string `json:"targetId"`
	TargetKind string `json:"targetKind"`
	Hostname   string `json:"hostname"`
	Status     string `json:"status"`
	Error      string `json:"error"`
}
