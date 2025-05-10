package planner

import (
	"bufio"
	"fmt"

	"github.com/quetzal-deploy/quetzal/internal/nix"
	"github.com/quetzal-deploy/quetzal/internal/steps"
)

type XSchedule struct {
	Period  int
	Timeout int
}

type StepStatus struct {
	Id     string
	Status string
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

type RequestPlus struct {
	Request steps.Request
	Period  int
	Timeout int
}

func CreateBuildPlan(hosts []nix.Host) steps.Step {
	hostNames := make([]string, 0)
	for _, host := range hosts {
		hostNames = append(hostNames, host.Name)
	}

	return steps.New().
		Description("build hosts").
		Action(&steps.Build{
			Hosts: hostNames,
		}).
		ExitOnFailure().
		Build()
}

func pushId(host nix.Host) string {
	return "push:" + host.Name
}

func deployId(host nix.Host) string {
	return "deploy:" + host.TargetHost
}

func CreateStepGetSudoPasswd() steps.Step {
	return steps.New().
		Description("Get sudo password").
		Action(&steps.GetSudoPasswd{}).
		DisableResume().
		Build()
}

func CreateStepSkip(skippedStep steps.Step) steps.Step {
	return steps.New().
		Description(skippedStep.Action.Name() + ": " + skippedStep.Description).
		Action(&steps.Skip{}).
		Build()
}

func CreateStepPush(host nix.Host) steps.Step {
	return steps.New().
		Id(pushId(host)).
		Description(fmt.Sprintf("push to %s", host.Name)).
		Action(&steps.Push{
			Host: host.Name,
		}).
		ExitOnFailure().
		Build()
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

func CreateStepChecks(hint string, host nix.Host, localCommands []CommandPlus, remoteCommands []CommandPlus, localRequests []RequestPlus, remoteRequests []RequestPlus) steps.Step {
	gate := CreateStepGate(hint + " for " + host.Name)
	gate.Id = hint + ":" + host.Name

	for _, commandPlus := range localCommands {
		step := CreateStepLocalCommand(host, commandPlus.Command)
		step.Timeout = commandPlus.Timeout
		step.RetryInterval = commandPlus.Period

		gate.Steps = append(gate.Steps, step)
	}

	for _, commandPlus := range remoteCommands {
		step := CreateStepRemoteCommand(host, commandPlus.Command)
		step.Timeout = commandPlus.Timeout
		step.RetryInterval = commandPlus.Period

		gate.Steps = append(gate.Steps, step)
	}

	for _, reqPlus := range localRequests {
		step := CreateStepLocalHttpRequest(host, reqPlus.Request)
		step.Timeout = reqPlus.Timeout
		step.RetryInterval = reqPlus.Period

		gate.Steps = append(gate.Steps, step)
	}

	for _, reqPlus := range remoteRequests {
		step := CreateStepRemoteHttpRequest(host, reqPlus.Request)
		step.Timeout = reqPlus.Timeout
		step.RetryInterval = reqPlus.Period

		gate.Steps = append(gate.Steps, step)

	}

	return gate
}

func CreateStepReboot(host nix.Host) steps.Step {
	return steps.New().
		Description("reboot " + host.Name).
		Action(&steps.Reboot{Host: host.Name}).
		Build()
}

// FIXME: change to remote command
func CreateStepIsOnline(host nix.Host) steps.Step {
	return steps.New().
		Description("test if " + host.Name + " is online").
		Action(&steps.IsOnline{Host: host.Name}).
		Build()
}

func CreateStepWaitForOnline(host nix.Host) steps.Step {
	step := CreateStepIsOnline(host)
	step.OnFailure = "retry"
	step.RetryInterval = 2

	return step
}

func CreateStepRebootAndWait(host nix.Host) steps.Step {
	return steps.New().
		Description(fmt.Sprintf("reboot '%s' and wait for it to come online", host.Name)).
		AddSequentialSteps(
			CreateStepReboot(host),
			CreateStepWaitForOnline(host)).Build()
}

func CreateStepGate(description string) steps.Step {
	return steps.New().
		Description(description).
		Action(&steps.Gate{}).
		RetryOnFailure().
		Parallel().
		Build()
}

// commands

func createStepCommand(location string, host nix.Host, command Command) steps.Step {
	step := steps.New().
		Description(command.Description)

	switch location {
	case "local":
		step.Action(&steps.LocalCommand{
			Command: command.Command,
			Timeout: 0,
		})
	case "remote":
		step.Action(&steps.RemoteCommand{
			Command: command.Command,
			Timeout: 0,
		})
	default:
		panic("Unknown location type")
	}

	return step.Build()
}

func CreateStepLocalCommand(host nix.Host, command Command) steps.Step {
	return createStepCommand("local", host, command)
}

func CreateStepRemoteCommand(host nix.Host, command Command) steps.Step {
	return createStepCommand("remote", host, command)
}

// HTTP requests

// FIXME: Get rid of the health check types
func CreateStepLocalHttpRequest(host nix.Host, req steps.Request) steps.Step {
	return steps.New().
		Description(req.Description).
		Action(&steps.LocalRequest{
			Request: req,
			Timeout: 0,
		}).
		Build()
}

// FIXME: Get rid of the health check types
func CreateStepRemoteHttpRequest(host nix.Host, req steps.Request) steps.Step {
	return steps.New().
		Description(req.Description).
		Action(&steps.RemoteRequest{
			Request: req,
			Timeout: 0}).
		Build()
}

// deploy wrappers

func createStepDeploy(deployAction steps.Action, host nix.Host, dependencies ...steps.Step) steps.Step {
	return steps.New().
		Id(deployId(host)).
		Description("deploy " + host.Name).
		Action(deployAction).
		DoNothingOnFailure().
		AddDependenciesSteps(dependencies...).
		Build()
}

func CreateStepDeployBoot(host nix.Host, dependencies ...steps.Step) steps.Step {
	return createStepDeploy(&steps.DeployBoot{Host: host.Name}, host, dependencies...)
}

func CreateStepDeployDryActivate(host nix.Host, dependencies ...steps.Step) steps.Step {
	return createStepDeploy(&steps.DeployDryActivate{Host: host.Name}, host, dependencies...)
}

func CreateStepDeploySwitch(host nix.Host, dependencies ...steps.Step) steps.Step {
	return createStepDeploy(&steps.DeploySwitch{Host: host.Name}, host, dependencies...)
}

func CreateStepDeployTest(host nix.Host, dependencies ...steps.Step) steps.Step {
	return createStepDeploy(&steps.DeployTest{Host: host.Name}, host, dependencies...)
}

// dot file output

func WriteDotFile(writer *bufio.Writer, plan steps.Step) {
	writer.WriteString("digraph G {\n")
	// writer.WriteString("\tlayout=fdp;")
	writer.WriteString("\trankdir=LR;")
	writer.WriteString("\tranksep=0.8;")
	defer writer.WriteString("}\n")

	CreateDotBla(writer, plan)
}

func CreateDotBla(writer *bufio.Writer, step steps.Step) {
	fmt.Println(step.Description)

	switch step.Action.Name() {
	case "":
		fallthrough
	case "none":
		writer.WriteString(fmt.Sprintf("\t\"%s\"[label = \"<f0> wrapper | <f1> %s\", shape=record, color=grey64, fontcolor=grey64, style=\"rounded,dashed\"]\n", step.Id, step.Description))
	case "skip":
		writer.WriteString(fmt.Sprintf("\t\"%s\"[label = \"<f0> skipped | <f1> %s\", shape=record, color=grey64, style=\"rounded,dashed\"]\n", step.Id, step.Description))
	case "build":
		hostsByName := step.Action.(*steps.Build).Hosts

		writer.WriteString(fmt.Sprintf("\t\"%s\"[label = \"<f0> build | <f1> %s", step.Id, step.Description))
		for i, host := range hostsByName {
			writer.WriteString(fmt.Sprintf(" | <f%d> %s", i+2, host))
		}
		writer.WriteString("\", shape=record, style=rounded]\n")
	default:
		writer.WriteString(fmt.Sprintf("\t\"%s\"[label = \"<f0> %s | <f1> %s\", shape=record, style=rounded]\n", step.Id, step.Action.Name(), step.Description))
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
