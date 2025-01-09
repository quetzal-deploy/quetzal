package planner

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/DBCDK/morph/nix"
	"github.com/google/uuid"
)

type Plan struct {
	Steps      []Step
	StepStatus map[string]string // FIXME: Step ID's - should probably be some custom type with more data about the step that ran (like errors)
	StepDone   map[string]bool   // FIXME: Probably don't need both this and StepStatus
	Cache      map[string]string // FIXME: ? Data written to the cache during processing // the StepData and cacheWriter stuff should probably be formalized somehow and moved away from the specific implementation
}

type Step struct {
	Id          string   `json:"id"`
	Description string   `json:"description"`
	ActionName  string   `json:"action"`
	Action      Action   `json:"-"`
	Parallel    bool     `json:"parallel"`
	OnFailure   string   `json:"on-failure"` // retry, exit, ignore
	Steps       []Step   `json:"steps"`
	DependsOn   []string `json:"dependencies"`
	CanResume   bool     `json:"can-resume"`
}

func (step Step) MarshalJSON() ([]byte, error) {
	type StepAlias Step

	switch step.ActionName {
	case NoAction{}.Name():
		fallthrough
	case Gate{}.Name():
		fallthrough
	case "wrapper": // FIXME: Either delete wrapper or create it as proper action
		return json.Marshal(StepAlias(step))

	case Build{}.Name():
		return json.Marshal(struct {
			StepAlias
			Build
		}{
			StepAlias: StepAlias(step),
			Build:     step.Action.(Build),
		})

	case Push{}.Name():
		return json.Marshal(struct {
			StepAlias
			Push
		}{
			StepAlias: StepAlias(step),
			Push:      step.Action.(Push),
		})

	case RepeatUntilSuccess{}.Name():
		return json.Marshal(struct {
			StepAlias
			RepeatUntilSuccess
		}{
			StepAlias:          StepAlias(step),
			RepeatUntilSuccess: step.Action.(RepeatUntilSuccess),
		})

	case DeployBoot{}.Name():
		return json.Marshal(struct {
			StepAlias
			DeployBoot
		}{
			StepAlias:  StepAlias(step),
			DeployBoot: step.Action.(DeployBoot),
		})

	case DeployDryActivate{}.Name():
		return json.Marshal(struct {
			StepAlias
			DeployDryActivate
		}{
			StepAlias:         StepAlias(step),
			DeployDryActivate: step.Action.(DeployDryActivate),
		})

	case DeploySwitch{}.Name():
		return json.Marshal(struct {
			StepAlias
			DeploySwitch
		}{
			StepAlias:    StepAlias(step),
			DeploySwitch: step.Action.(DeploySwitch),
		})

	case DeployTest{}.Name():
		return json.Marshal(struct {
			StepAlias
			DeployTest
		}{
			StepAlias:  StepAlias(step),
			DeployTest: step.Action.(DeployTest),
		})

	case LocalCommandAction{}.Name():
		return json.Marshal(struct {
			StepAlias
			LocalCommandAction
		}{
			StepAlias:          StepAlias(step),
			LocalCommandAction: step.Action.(LocalCommandAction),
		})

	case RemoteCommandAction{}.Name():
		return json.Marshal(struct {
			StepAlias
			RemoteCommandAction
		}{
			StepAlias:           StepAlias(step),
			RemoteCommandAction: step.Action.(RemoteCommandAction),
		})

	case LocalRequestAction{}.Name():
		return json.Marshal(struct {
			StepAlias
			LocalRequestAction
		}{
			StepAlias:          StepAlias(step),
			LocalRequestAction: step.Action.(LocalRequestAction),
		})

	case RemoteRequestAction{}.Name():
		return json.Marshal(struct {
			StepAlias
			RemoteRequestAction
		}{
			StepAlias:           StepAlias(step),
			RemoteRequestAction: step.Action.(RemoteRequestAction),
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
	case "none":
		fallthrough
	case "gate":
		fallthrough
	case "wrapper":
		// do nothing
		fmt.Println("action: none")

	case "build":
		fmt.Println("action: build")
		var build Build
		err = json.Unmarshal(b, &build)
		if err != nil {
			return err
		}

		step.Action = build

	case "push":
		fmt.Println("action: push")
		var push Push
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

type Action interface {
	Name() string
	ToMap() map[string]interface{}
}

type NoAction struct{}
type Gate struct{}

type EvalDeployment struct {
	Deployment string `json:"deployment"`
}

type Build struct {
	Hosts []string `json:"hosts"`
}

type Push struct {
	Host string `json:"host"`
}

type Reboot struct {
	Host string `json:"host"`
}

type IsOnline struct {
	Host string `json:"host"`
}

type RepeatUntilSuccess struct {
	Period  int `json:"period"`
	Timeout int `json:"timeout"`
}

type ActionWithOneHost struct {
	Host string `json:"host"`
}

type XSchedule struct {
	Period  int
	Timeout int
}

type LocalRequestAction struct {
	Request Request `json:"request"`
	Timeout int     `json:"timeout"`
}

type RemoteRequestAction struct {
	Request Request `json:"request"`
	Timeout int     `json:"timeout"`
}

type LocalCommandAction struct {
	Command []string `json:"command"`
	Timeout int      `json:"timeout"`
}

type RemoteCommandAction struct {
	Command []string `json:"command"`
	Timeout int      `json:"timeout"`
}

func (a ActionWithOneHost) ToMap() map[string]interface{} {
	return justAHost(a.Host)
}

type DeployBoot struct{ ActionWithOneHost }
type DeployDryActivate struct{ ActionWithOneHost }
type DeploySwitch struct{ ActionWithOneHost }
type DeployTest struct{ ActionWithOneHost }

type GetSudoPasswd struct{}

func (_ NoAction) Name() string            { return "none" }
func (_ Gate) Name() string                { return "gate" }
func (_ GetSudoPasswd) Name() string       { return "get-sudo-password" }
func (_ EvalDeployment) Name() string      { return "eval-deployment" }
func (_ Build) Name() string               { return "build" }
func (_ Push) Name() string                { return "push" }
func (_ Reboot) Name() string              { return "reboot" }
func (_ IsOnline) Name() string            { return "is-online" }
func (_ RepeatUntilSuccess) Name() string  { return "repeat-until-success" }
func (_ DeployBoot) Name() string          { return "deploy-boot" }
func (_ DeployDryActivate) Name() string   { return "deploy-dry-activate" }
func (_ DeploySwitch) Name() string        { return "deploy-switch" }
func (_ DeployTest) Name() string          { return "deploy-test" }
func (_ LocalRequestAction) Name() string  { return "local-request" }
func (_ RemoteRequestAction) Name() string { return "remote-request" }
func (_ LocalCommandAction) Name() string  { return "local-command" }
func (_ RemoteCommandAction) Name() string { return "remote-command" }

func justNothing() map[string]interface{} {
	obj := make(map[string]interface{})
	return obj
}

func justAHost(host string) map[string]interface{} {
	obj := make(map[string]interface{})
	obj["host"] = host
	return obj
}

func (x EvalDeployment) ToMap() map[string]interface{} {
	return justNothing()
}

func (x LocalCommandAction) ToMap() map[string]interface{} {
	return justNothing()
}

func (x RemoteCommandAction) ToMap() map[string]interface{} {
	return justNothing()
}

func (x LocalRequestAction) ToMap() map[string]interface{} {
	return justNothing()
}

func (x RemoteRequestAction) ToMap() map[string]interface{} {
	return justNothing()
}

func (a Gate) ToMap() map[string]interface{} {
	return justNothing()
}

func (a NoAction) ToMap() map[string]interface{} {
	return justNothing()
}

func (a GetSudoPasswd) ToMap() map[string]interface{} {
	return justNothing()
}

func (a Build) ToMap() map[string]interface{} {
	obj := make(map[string]interface{})
	obj["hosts"] = a.Hosts
	return obj
}

func (a Push) ToMap() map[string]interface{} {
	return justAHost(a.Host)
}

func (a Reboot) ToMap() map[string]interface{} {
	return justAHost(a.Host)
}

func (a IsOnline) ToMap() map[string]interface{} {
	return justAHost(a.Host)
}

func (a RepeatUntilSuccess) ToMap() map[string]interface{} {
	return justNothing()
}

type StepStatus struct {
	Id     string
	Status string
}

type StepData struct {
	Key   string
	Value string
}

type Command struct {
	Description string
	Command     []string
}

type CommandPlus struct {
	Command Command
	Period  int
	Timeout int
}

type Request struct {
	Description string            `json:"description"`
	Headers     map[string]string `json:"headers"`
	Host        *string           `json:"host"`
	InsecureSSL bool              `json:"insecureSSL"`
	Path        string            `json:"path"`
	Port        int               `json:"port"`
	Scheme      string            `json:"scheme"`
}

type RequestPlus struct {
	Request Request
	Period  int
	Timeout int
}

func CreateStep(description string, actionName string, action Action, parallel bool, steps []Step, onFailure string, dependencies []string) Step {
	step := Step{
		Id:          uuid.New().String(),
		Description: description,
		ActionName:  actionName,
		Action:      action,
		Parallel:    parallel,
		Steps:       steps,
		OnFailure:   onFailure,
		DependsOn:   dependencies,
		CanResume:   true,
	}

	return step
}

func EmptyStep() Step {
	step := Step{
		Id:          uuid.New().String(),
		Description: "",
		ActionName:  "none",
		Action:      nil,
		Parallel:    false,
		Steps:       make([]Step, 0),
		OnFailure:   "",
		DependsOn:   make([]string, 0),
		CanResume:   true,
	}

	return step
}

func AddSteps(plan Step, steps ...Step) Step {
	plan.Steps = append(plan.Steps, steps...)

	// for _, step := range steps {
	// 	plan.DependsOn = append(plan.DependsOn, step.Id)
	// 	plan.Steps = append(plan.Steps, step)
	// }

	return plan
}

func AddStepsSeq(plan Step, steps ...Step) Step {
	for _, step := range steps {
		if len(plan.Steps) > 0 {
			// If tthere's existing steps, get the ID of the last one and add it as dependency to the current one
			step.DependsOn = append(step.DependsOn, plan.Steps[len(plan.Steps)-1].Id)
		}

		plan.Steps = append(plan.Steps, step)
	}

	return plan
}

func EmptySteps() []Step {
	return make([]Step, 0)
}

func MakeDependencies(dependencies ...string) []string {
	deps := make([]string, 0)
	deps = append(deps, dependencies...)

	return deps
}

func CreateBuildPlan(hosts []nix.Host) Step {
	hostNames := make([]string, 0)
	for _, host := range hosts {
		hostNames = append(hostNames, host.Name)
	}

	action := Build{
		Hosts: hostNames,
	}

	return CreateStep("build hosts", "build", action, false, EmptySteps(), "exit", make([]string, 0))
}

func pushId(host nix.Host) string {
	return "push:" + host.TargetHost
}

func deployId(host nix.Host) string {
	return "deploy:" + host.TargetHost
}

func CreateStepGetSudoPasswd() Step {
	step := EmptyStep()
	step.Description = "Get sudo password"
	step.ActionName = "get-sudo-passwd"
	step.Action = GetSudoPasswd{}
	step.CanResume = false

	return step
}

func CreateStepSkip(skippedStep Step) Step {
	step := EmptyStep()
	step.Description = skippedStep.ActionName + ": " + skippedStep.Description
	step.ActionName = "skip"

	return step
}

func CreateStepPush(host nix.Host) Step {
	push := Push{
		Host: host.Name,
	}

	step := CreateStep(fmt.Sprintf("push to %s", host.Name), "push", push, true, EmptySteps(), "exit", make([]string, 0))
	step.Id = pushId(host)

	return step
}

func CreatePushPlan(buildId string, hosts []nix.Host) Step {
	pushParent := CreateStep("push to hosts", "none", NoAction{}, true, EmptySteps(), "exit", MakeDependencies(buildId))

	for _, host := range hosts {
		pushParent = AddSteps(
			pushParent,
			CreateStepPush(host),
		)
	}

	return pushParent
}

// func CreateStepHealthChecks(host nix.Host, checks healthchecks.HealthChecks) Step {
// 	gate := CreateStepGate("healthchecks for " + host.Name)

// 	for _, check := range checks.Cmd {
// 		req := CreateStepRemoteCommand(host, check)
// 		runner := CreateStepRepeatUntilSuccess(check.Period, check.Timeout)
// 		runner = AddSteps(runner, req)

// 		gate.Steps = append(gate.Steps, runner)
// 	}

// 	for _, check := range checks.Http {
// 		req := CreateStepRemoteHttpRequest(host, check)
// 		runner := CreateStepRepeatUntilSuccess(check.Period, check.Timeout)
// 		runner = AddSteps(runner, req)

// 		gate.Steps = append(gate.Steps, runner)
// 	}

// 	return gate
// }

func CreateStepChecks(host nix.Host, localCommands []CommandPlus, remoteCommands []CommandPlus, localRequests []RequestPlus, remoteRequests []RequestPlus) Step {
	gate := CreateStepGate("healthchecks for " + host.Name)

	for _, commandPlus := range localCommands {
		step := CreateStepLocalCommand(host, commandPlus.Command)
		runner := CreateStepRepeatUntilSuccess(commandPlus.Period, commandPlus.Timeout)
		runner = AddSteps(runner, step)

		gate.Steps = append(gate.Steps, runner)
	}

	for _, commandPlus := range remoteCommands {
		step := CreateStepRemoteCommand(host, commandPlus.Command)
		runner := CreateStepRepeatUntilSuccess(commandPlus.Period, commandPlus.Timeout)
		runner = AddSteps(runner, step)

		gate.Steps = append(gate.Steps, runner)
	}

	for _, reqPlus := range localRequests {
		step := CreateStepLocalHttpRequest(host, reqPlus.Request)
		runner := CreateStepRepeatUntilSuccess(reqPlus.Period, reqPlus.Timeout)
		runner = AddSteps(runner, step)

		gate.Steps = append(gate.Steps, runner)
	}

	for _, reqPlus := range remoteRequests {
		step := CreateStepRemoteHttpRequest(host, reqPlus.Request)
		runner := CreateStepRepeatUntilSuccess(reqPlus.Period, reqPlus.Timeout)
		runner = AddSteps(runner, step)

		gate.Steps = append(gate.Steps, runner)

	}

	return gate
}

func CreateStepReboot(host nix.Host) Step {
	step := EmptyStep()
	step.Description = "reboot " + host.Name
	step.ActionName = "reboot"
	step.Action = Reboot{Host: host.Name}

	return step
}

// FIXME: change to remote command
func CreateStepIsOnline(host nix.Host) Step {
	step := EmptyStep()
	step.ActionName = "is-online"
	step.Action = IsOnline{Host: host.Name}
	step.Description = "test if " + host.Name + " is online"

	command := Command{
		Description: "check host is online",
		Command:     []string{"/bin/sh", "-c", "true"},
	}

	CreateStepRemoteCommand(host, command)

	return step
}

func CreateStepWaitForOnline(host nix.Host) Step {
	step := EmptyStep()
	step.ActionName = "wait-for-online"
	step.Action = NoAction{}
	step.Description = fmt.Sprintf("Wait for %s to come online", host.Name)

	timeout := 5
	period := 10

	wait := CreateStepRepeatUntilSuccess(timeout, period)
	wait = AddSteps(wait, CreateStepIsOnline(host))

	step = AddSteps(step, wait)

	return step
}

func CreateStepRepeatUntilSuccess(period int, timeout int) Step {
	step := EmptyStep()
	step.ActionName = "repeat-until-success"
	step.Action = RepeatUntilSuccess{
		Period:  timeout, // FIXME: ???
		Timeout: timeout,
	}
	step.Description = fmt.Sprintf("period=%d timeout=%d", period, timeout)

	return step
}

func CreateStepGate(description string) Step {
	step := EmptyStep()
	step.ActionName = "gate"
	step.Action = Gate{}
	step.Description = description
	step.Parallel = true
	step.OnFailure = "retry"

	return step
}

// commands

func createStepCommand(location string, host nix.Host, command Command) Step {
	step := EmptyStep()

	switch location {
	case "local":
		step.ActionName = "local-command"
	case "remote":
		step.ActionName = "remote-command"
	default:
		panic("Unknown location type")
	}

	step.Description = command.Description
	step.Action = RemoteCommandAction{
		Command: command.Command,
		Timeout: 0,
	}

	return step
}

func CreateStepLocalCommand(host nix.Host, command Command) Step {
	return createStepCommand("local", host, command)
}

func CreateStepRemoteCommand(host nix.Host, command Command) Step {
	return createStepCommand("remote", host, command)
}

// HTTP requests

// FIXME: Get rid of the health check types
func CreateStepLocalHttpRequest(host nix.Host, req Request) Step {
	step := EmptyStep()

	step.Description = req.Description
	step.Action = LocalRequestAction{
		Request: req,
		Timeout: 0,
	}
	step.ActionName = step.Action.Name()

	return step
}

// FIXME: Get rid of the health check types
func CreateStepRemoteHttpRequest(host nix.Host, req Request) Step {
	step := EmptyStep()

	step.Description = req.Description
	step.Action = RemoteRequestAction{
		Request: req,
		Timeout: 0,
	}
	step.ActionName = step.Action.Name()

	return step
}

// deploy wrappers

func createStepDeploy(deployAction Action, host nix.Host, dependencies ...Step) Step {
	step := EmptyStep()
	step.Id = deployId(host)
	step.Description = "deploy " + host.Name
	step.ActionName = deployAction.Name()
	step.Action = deployAction
	step.OnFailure = ""

	for _, dependency := range dependencies {
		step.DependsOn = append(step.DependsOn, dependency.Id)
	}

	return step
}

func CreateStepDeployBoot(host nix.Host, dependencies ...Step) Step {
	return createStepDeploy(DeployBoot{ActionWithOneHost{Host: host.Name}}, host, dependencies...)
}

func CreateStepDeployDryActivate(host nix.Host, dependencies ...Step) Step {
	return createStepDeploy(DeployDryActivate{ActionWithOneHost{Host: host.Name}}, host, dependencies...)
}

func CreateStepDeploySwitch(host nix.Host, dependencies ...Step) Step {
	return createStepDeploy(DeploySwitch{ActionWithOneHost{Host: host.Name}}, host, dependencies...)
}

func CreateStepDeployTest(host nix.Host, dependencies ...Step) Step {
	return createStepDeploy(DeployTest{ActionWithOneHost{Host: host.Name}}, host, dependencies...)
}

// dot file output

func WriteDotFile(writer *bufio.Writer, plan Step) {
	writer.WriteString("digraph G {\n")
	// writer.WriteString("\tlayout=fdp;")
	writer.WriteString("\trankdir=LR;")
	writer.WriteString("\tranksep=0.8;")
	defer writer.WriteString("}\n")

	CreateDotBla(writer, plan)
}

func CreateDotBla(writer *bufio.Writer, step Step) {
	fmt.Println(step.Description)

	switch step.ActionName {
	case "":
		fallthrough
	case "none":
		writer.WriteString(fmt.Sprintf("\t\"%s\"[label = \"<f0> wrapper | <f1> %s\", shape=record, color=grey64, fontcolor=grey64, style=\"rounded,dashed\"]\n", step.Id, step.Description))
	case "skip":
		writer.WriteString(fmt.Sprintf("\t\"%s\"[label = \"<f0> skipped | <f1> %s\", shape=record, color=grey64, style=\"rounded,dashed\"]\n", step.Id, step.Description))
	case "build":
		hostsByName := step.Action.(Build).Hosts

		writer.WriteString(fmt.Sprintf("\t\"%s\"[label = \"<f0> build | <f1> %s", step.Id, step.Description))
		for i, host := range hostsByName {
			writer.WriteString(fmt.Sprintf(" | <f%d> %s", i+2, host))
		}
		writer.WriteString("\", shape=record, style=rounded]\n")
	default:
		writer.WriteString(fmt.Sprintf("\t\"%s\"[label = \"<f0> %s | <f1> %s\", shape=record, style=rounded]\n", step.Id, step.ActionName, step.Description))
	}

	for _, dependency := range step.DependsOn {
		writer.WriteString(fmt.Sprintf("\t\"%s\" -> \"%s\" [dir=back, color=deepskyblue, penwidth=1.0, style=dashed];\n", dependency, step.Id))
	}

	for _, subStep := range step.Steps {
		// writer.WriteString(fmt.Sprintf("\t\"%s\" -> \"%s\" [style=dotted];\n", step.Id, subStep.Id))
		writer.WriteString(fmt.Sprintf("\t\"%s\" -> \"%s\" [color=grey64, penwidth=0.8];\n", step.Id, subStep.Id))
		CreateDotBla(writer, subStep)
	}
}
