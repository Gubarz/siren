package journal

type VerbPolicy int

const (
	VerbRecord VerbPolicy = iota
	VerbDrop
)

var droppedVerbs = map[string]struct{}{
	"GetSessions":    {},
	"GetBeacons":     {},
	"GetJobs":        {},
	"GetVersion":     {},
	"GetBeaconTask":  {},
	"GetBeaconTasks": {},
	"Events":         {},
	"ClientLog":      {},
	"TunnelData":     {},
}

func ClassifyVerb(method string) VerbPolicy {
	if _, ok := droppedVerbs[method]; ok {
		return VerbDrop
	}
	return VerbRecord
}
