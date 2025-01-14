package planner

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/DBCDK/morph/actions"
)

// The following MarshalJSON contains a lot of code that looks
// like it could be easily compacted. Ideas welcome.
// Type parameters can't be embedded, otherwise something like
// this would help cut a lot of code:
//
// func x[A actions.Action](step Step) ([]byte, error) {
//   type StepAlias Step
//   return json.Marshal(struct {
//     StepAlias
//     A // I'm sorry Dave, I'm afraid I can't do that
//   }{
//     StepAlias: StepAlias(step),
//     A:         step.Action.(A),
//   })
// }

func (step Step) MarshalJSON() ([]byte, error) {
	type StepAlias Step

	switch step.ActionName {
	case actions.None{}.Name():
		fallthrough
	case actions.Gate{}.Name():
		fallthrough
	case actions.Wrapper{}.Name():
		return json.Marshal(StepAlias(step))

	case actions.Build{}.Name():
		return json.Marshal(struct {
			StepAlias
			actions.Build
		}{
			StepAlias: StepAlias(step),
			Build:     step.Action.(actions.Build),
		})

	case actions.Push{}.Name():
		return json.Marshal(struct {
			StepAlias
			actions.Push
		}{
			StepAlias: StepAlias(step),
			Push:      step.Action.(actions.Push),
		})

	case actions.DeployBoot{}.Name():
		return json.Marshal(struct {
			StepAlias
			actions.DeployBoot
		}{
			StepAlias:  StepAlias(step),
			DeployBoot: step.Action.(actions.DeployBoot),
		})

	case actions.DeployDryActivate{}.Name():
		return json.Marshal(struct {
			StepAlias
			actions.DeployDryActivate
		}{
			StepAlias:         StepAlias(step),
			DeployDryActivate: step.Action.(actions.DeployDryActivate),
		})

	case actions.DeploySwitch{}.Name():
		return json.Marshal(struct {
			StepAlias
			actions.DeploySwitch
		}{
			StepAlias:    StepAlias(step),
			DeploySwitch: step.Action.(actions.DeploySwitch),
		})

	case actions.DeployTest{}.Name():
		return json.Marshal(struct {
			StepAlias
			actions.DeployTest
		}{
			StepAlias:  StepAlias(step),
			DeployTest: step.Action.(actions.DeployTest),
		})

	case actions.IsOnline{}.Name():
		return json.Marshal(struct {
			StepAlias
			actions.IsOnline
		}{
			StepAlias: StepAlias(step),
			IsOnline:  step.Action.(actions.IsOnline),
		})

	case actions.Reboot{}.Name():
		return json.Marshal(struct {
			StepAlias
			actions.Reboot
		}{
			StepAlias: StepAlias(step),
			Reboot:    step.Action.(actions.Reboot),
		})

	case actions.LocalCommand{}.Name():
		return json.Marshal(struct {
			StepAlias
			actions.LocalCommand
		}{
			StepAlias:    StepAlias(step),
			LocalCommand: step.Action.(actions.LocalCommand),
		})

	case actions.RemoteCommand{}.Name():
		return json.Marshal(struct {
			StepAlias
			actions.RemoteCommand
		}{
			StepAlias:     StepAlias(step),
			RemoteCommand: step.Action.(actions.RemoteCommand),
		})

	case actions.LocalRequest{}.Name():
		return json.Marshal(struct {
			StepAlias
			actions.LocalRequest
		}{
			StepAlias:    StepAlias(step),
			LocalRequest: step.Action.(actions.LocalRequest),
		})

	case actions.RemoteRequest{}.Name():
		return json.Marshal(struct {
			StepAlias
			actions.RemoteRequest
		}{
			StepAlias:     StepAlias(step),
			RemoteRequest: step.Action.(actions.RemoteRequest),
		})

	default:
		return nil, errors.New("unmarshall: unknown action: " + step.ActionName)
	}
}

func (step *Step) UnmarshalJSON(b []byte) error {
	// A step is unmarshalled twice:
	// 1) into an alias for the Step struct
	// 2) into the type matching the action name of the step
	// (2) is then added as the action to the step

	// This alias keeps the original methods implemented
	// on Step, in this case the default UnmarshalJSON method.

	type StepAlias Step

	// (1) Unmarshal everything in to the step alias, and assign it to *step
	// *step will then be populated according to the Step-struct but without the Action

	// Safe defaults for Step
	step_ := StepAlias{
		ActionName: "none",
		Parallel:   false,
		CanResume:  false,
		DependsOn:  make([]string, 0),
		Steps:      make([]Step, 0),
	}

	err := json.Unmarshal(b, &step_)
	if err != nil {
		return err
	}
	*step = Step(step_)

	//_ = map[string]func() interface{}{
	//	"build": func() interface{} { return &Build{} },
	//	"push":  func() interface{} { return &Push{} },
	//}

	// (2) Unmarshal the same into the corresponding Action and
	// add it to the step
	switch step.ActionName {
	case actions.None{}.Name():
		fallthrough
	case actions.Gate{}.Name():
		fallthrough
	case actions.Wrapper{}.Name():
		// do nothing
		fmt.Println("action: none")

	case actions.Build{}.Name():
		fmt.Println("action: build")
		var build actions.Build
		err = json.Unmarshal(b, &build)
		if err != nil {
			return err
		}

		step.Action = build

	case actions.Push{}.Name():
		fmt.Println("action: push")
		var push actions.Push
		err = json.Unmarshal(b, &push)
		if err != nil {
			return err
		}

		step.Action = push

	default:
		return errors.New("unmarshal: unknown action: " + step.ActionName)
	}

	return nil
}
