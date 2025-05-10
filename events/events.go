package events

import (
	"github.com/quetzal-deploy/quetzal/steps"
)

type Event interface {
	// Name() string
}

type Debug struct {
	Data string
}

type Log struct { // FIXME: Get rid of this. Stop sending all stdout/stederr as events
	Data string
}

func (e Log) Name() string { return "log" }

type RegisterPlan struct {
	Plan steps.Step
}

func (e RegisterPlan) Name() string { return "register_plan" }

type RegisterStep struct {
	Step steps.Step
}

func (e RegisterStep) Name() string { return "register_step" }

type StepUpdate struct { // TODO: Rename -> StepStatus
	StepId string
	State  string
}

func (e StepUpdate) Name() string { return "step_update" }

// FIXME: Merge this with LogEvent
type StepLog struct {
	StepId string
	Data   string
}

type StepStatus struct { // TODO: Rename -> StepBlocked
	Step      steps.Step
	BlockedBy []string
}

type QueueStatus struct {
	Queue []StepStatus
}

func (e QueueStatus) Name() string { return "queue_status" }

type Pause struct{}
type Unpause struct{} // Maybe this should be called "Running" instead
type StatePaused struct{}
type StateUnpaused struct{} // Maybe this should be called "Running" instead

func (e Pause) Name() string         { return "pause" }
func (e Unpause) Name() string       { return "unpause" }
func (e StatePaused) Name() string   { return "state_paused" }
func (e StateUnpaused) Name() string { return "state_unpaused" }

// https://stackoverflow.com/a/61916765
// type TypeSwitch struct {
// 	Type string `json:"type"`
// }

// type Event2 struct {
// 	TypeSwitch
// 	*StepQueued
// }

// func (t *Event2) UnmarshalJSON(data []byte) error {
// 	if err := json.Unmarshal(data, &t.TypeSwitch); err != nil {
// 		return err
// 	}

// 	switch t.Type {
// 	case "step_queued":
// 		t.StepQueued = &StepQueued{}
// 		return json.Unmarshal(data, t.StepQueued)

// 	default:
// 		return fmt.Errorf("unknown type: %q", t.Type)
// 	}
// }

// func (t Event2) MarshalJSON() ([]byte, error) {
// 	switch t.Type {
// 	case "step_queued":
// 		return json.Marshal(struct {
// 			TypeSwitch
// 			*StepQueued
// 		}{
// 			t.TypeSwitch,
// 			t.StepQueued,
// 		})
// 	default:
// 		// return json.Marshal(t.TypeSwitch)
// 		return nil, fmt.Errorf("unknown type: %q", t.Type)
// 	}
// }
