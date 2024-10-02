package planner

import (
	"bufio"
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
	Id          string
	Description string
	Action      string
	Parallel    bool
	OnFailure   string // retry, exit, ignore
	Steps       []Step
	Options     map[string]interface{}
	DependsOn   []string
	Host        *nix.Host
	CanResume   bool
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
	Description string
	Headers     map[string]string
	Host        *string
	InsecureSSL bool
	Path        string
	Port        int
	Scheme      string
}

type RequestPlus struct {
	Request Request
	Period  int
	Timeout int
}

func CreateStep(description string, action string, parallel bool, steps []Step, onFailure string, options map[string]interface{}, dependencies []string) Step {
	step := Step{
		Id:          uuid.New().String(),
		Description: description,
		Action:      action,
		Parallel:    parallel,
		Steps:       steps,
		OnFailure:   onFailure,
		Options:     options,
		DependsOn:   dependencies,
		Host:        nil,
		CanResume:   true,
	}

	return step
}

func EmptyStep() Step {
	step := Step{
		Id:          uuid.New().String(),
		Description: "",
		Action:      "none",
		Parallel:    false,
		Steps:       make([]Step, 0),
		OnFailure:   "",
		Options:     make(map[string]interface{}, 0),
		DependsOn:   make([]string, 0),
		Host:        nil,
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

func EmptyOptions() map[string]interface{} {
	return make(map[string]interface{}, 0)
}

func MakeDependencies(dependencies ...string) []string {
	deps := make([]string, 0)
	deps = append(deps, dependencies...)

	return deps
}

func CreateBuildPlan(hosts []nix.Host) Step {
	options := EmptyOptions()

	optionsHosts := make([]string, 0)
	for _, host := range hosts {
		optionsHosts = append(optionsHosts, host.Name)
	}

	options["hosts"] = optionsHosts

	return CreateStep("build hosts", "build", false, EmptySteps(), "exit", options, make([]string, 0))
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
	step.Action = "get-sudo-passwd"
	step.CanResume = false

	return step
}

func CreateStepSkip(skippedStep Step) Step {
	step := EmptyStep()
	step.Description = skippedStep.Action + ": " + skippedStep.Description
	step.Action = "skip"

	return step
}

func CreateStepPush(host nix.Host) Step {
	options := EmptyOptions()

	step := CreateStep(fmt.Sprintf("push to %s", host.Name), "push", true, EmptySteps(), "exit", options, make([]string, 0))
	step.Id = pushId(host)
	step.Host = &host

	return step
}

func CreatePushPlan(buildId string, hosts []nix.Host) Step {
	options := EmptyOptions()

	pushParent := CreateStep("push to hosts", "none", true, EmptySteps(), "exit", options, MakeDependencies(buildId))

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
	step.Action = "reboot"

	step.Options["host"] = host // FIXME: What is actually needed here?

	return step
}

// FIXME: change to remote command
func CreateStepIsOnline(host nix.Host) Step {
	step := EmptyStep()
	step.Action = "is-online"
	step.Description = "test if " + host.Name + " is online"

	step.Options["host"] = host

	command := Command{
		Description: "check host is online",
		Command:     []string{"/bin/sh", "-c", "true"},
	}

	CreateStepRemoteCommand(host, command)

	return step
}

func CreateStepWaitForOnline(host nix.Host) Step {
	step := EmptyStep()
	step.Action = "wait-for-online"
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
	step.Action = "repeat-until-success"
	step.Description = fmt.Sprintf("period=%d timeout=%d", period, timeout)
	step.Options["period"] = timeout
	step.Options["timeout"] = timeout

	return step
}

func CreateStepGate(description string) Step {
	step := EmptyStep()
	step.Action = "gate"
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
		step.Action = "local-command"
	case "remote":
		step.Action = "remote-command"
	default:
		panic("Unknown location type")
	}

	step.Description = command.Description
	step.Host = &host

	step.Options["cmd"] = command.Command

	return step
}

func CreateStepLocalCommand(host nix.Host, command Command) Step {
	return createStepCommand("local", host, command)
}

func CreateStepRemoteCommand(host nix.Host, command Command) Step {
	return createStepCommand("remote", host, command)
}

// HTTP requests

func createStepHttpRequest(location string, host nix.Host, req Request) Step {
	step := EmptyStep()

	switch location {
	case "local":
		step.Action = "local-http"
	case "remote":
		step.Action = "remote-http"
	default:
		panic("Unknown location type")
	}

	step.Description = req.Description

	step.Options["headers"] = req.Headers
	step.Options["host"] = req.Host
	step.Options["insecure-ssl"] = req.InsecureSSL
	step.Options["path"] = req.Path
	step.Options["port"] = req.Port
	step.Options["scheme"] = req.Scheme

	return step
}

// FIXME: Get rid of the health check types
func CreateStepLocalHttpRequest(host nix.Host, req Request) Step {
	return createStepHttpRequest("local", host, req)
}

// FIXME: Get rid of the health check types
func CreateStepRemoteHttpRequest(host nix.Host, req Request) Step {
	return createStepHttpRequest("remote", host, req)
}

// deploy wrappers

func createStepDeploy(deployAction string, host nix.Host, dependencies ...Step) Step {
	step := EmptyStep()
	step.Id = deployId(host)
	step.Description = "deploy " + host.Name
	step.Action = deployAction
	step.OnFailure = ""
	step.Host = &host

	for _, dependency := range dependencies {
		step.DependsOn = append(step.DependsOn, dependency.Id)
	}

	return step
}

func CreateStepDeployBoot(host nix.Host, dependencies ...Step) Step {
	return createStepDeploy("boot", host, dependencies...)
}

func CreateStepDeployDryActivate(host nix.Host, dependencies ...Step) Step {
	return createStepDeploy("dry-activate", host, dependencies...)
}

func CreateStepDeploySwitch(host nix.Host, dependencies ...Step) Step {
	return createStepDeploy("switch", host, dependencies...)
}

func CreateStepDeployTest(host nix.Host, dependencies ...Step) Step {
	return createStepDeploy("test", host, dependencies...)
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

	switch step.Action {
	case "":
		fallthrough
	case "none":
		writer.WriteString(fmt.Sprintf("\t\"%s\"[label = \"<f0> wrapper | <f1> %s\", shape=record, color=grey64, fontcolor=grey64, style=\"rounded,dashed\"]\n", step.Id, step.Description))
	case "skip":
		writer.WriteString(fmt.Sprintf("\t\"%s\"[label = \"<f0> skipped | <f1> %s\", shape=record, color=grey64, style=\"rounded,dashed\"]\n", step.Id, step.Description))
	case "build":
		hostsByName := step.Options["hosts"].([]string)

		writer.WriteString(fmt.Sprintf("\t\"%s\"[label = \"<f0> build | <f1> %s", step.Id, step.Description))
		for i, host := range hostsByName {
			writer.WriteString(fmt.Sprintf(" | <f%d> %s", i+2, host))
		}
		writer.WriteString("\", shape=record, style=rounded]\n")
	default:
		writer.WriteString(fmt.Sprintf("\t\"%s\"[label = \"<f0> %s | <f1> %s\", shape=record, style=rounded]\n", step.Id, step.Action, step.Description))
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
