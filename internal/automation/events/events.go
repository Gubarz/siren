// Package events defines bus payloads shared by producers and triggers
// without coupling the automation packages to the services that emit them.
package events

type TaskResult struct {
	Verb       string
	TargetID   string
	TargetKind string
	Hostname   string
	Status     string
	Error      string
}
