package automation

type State struct {
	Rules   []AutomationRule `json:"rules"`
	History []AutomationRun  `json:"history"`
}
