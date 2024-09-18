package planner

import (
	"bufio"
	"fmt"

	"github.com/DBCDK/morph/healthchecks"
	"github.com/DBCDK/morph/nix"
	"github.com/google/uuid"
)

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
}

type StepStatus struct {
	Id     string
	Status string
}

type StepData struct {
	Key   string
	Value string
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
	options["to"] = host.Name

	step := CreateStep(fmt.Sprintf("push to %s", host.Name), "push", true, EmptySteps(), "exit", options, make([]string, 0))
	step.Id = pushId(host)

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

func CreateStepCommandCheck(host nix.Host, check healthchecks.CmdHealthCheck) Step {
	step := EmptyStep()
	step.Description = check.Description
	step.Action = "cmd-check"
	step.OnFailure = "retry"
	// step.DependsOn = append(step.DependsOn, pushId(host), deployId(host)) // FIXME: Stop hardcoding action
	// FIXME: Checks should depend on push when only pushing, but push AND deploy when actually deploying

	step.Options["cmd"] = check.Cmd
	step.Options["period"] = check.Period
	step.Options["timeout"] = check.Timeout

	return step
}

func CreateStepHttpCheck(host nix.Host, check healthchecks.HttpHealthCheck) Step {
	step := EmptyStep()
	step.Description = check.Description
	step.Action = "http-check"
	step.OnFailure = "retry"
	// step.DependsOn = append(step.DependsOn, pushId(host)) // FIXME: Stop hardcoding action

	step.Options["headers"] = check.Headers
	step.Options["host"] = check.Host
	step.Options["insecure-ssl"] = check.InsecureSSL
	step.Options["path"] = check.Path
	step.Options["port"] = check.Port
	step.Options["scheme"] = check.Scheme
	step.Options["period"] = check.Period
	step.Options["timeout"] = check.Timeout

	return step
}

func CreateStepHealthChecks(host nix.Host, checks healthchecks.HealthChecks) Step {
	step := EmptyStep()
	step.Description = "healthchecks for " + host.TargetHost
	step.Action = "healthchecks"
	// step.DependsOn = append(step.DependsOn, deployId(host))
	step.Parallel = true
	step.OnFailure = "retry" // all sub-steps also retry - should both do that, or only the parent? or only the children?

	for _, check := range checks.Cmd {
		step.Steps = append(step.Steps, CreateStepCommandCheck(host, check))
	}

	for _, check := range checks.Http {
		step.Steps = append(step.Steps, CreateStepHttpCheck(host, check))
	}

	return step
}

func CreateHealthCheckPlan(hosts []nix.Host) Step {
	plan := EmptyStep()
	plan.Description = "healthchecks"
	plan.Action = ""
	plan.Parallel = true
	plan.OnFailure = ""

	for _, host := range hosts {
		step := CreateStepHealthChecks(host, host.HealthChecks)
		plan = AddSteps(plan, step)
	}

	return plan
}

func createStepDeploy(deployAction string, host nix.Host, dependencies ...Step) Step {
	step := EmptyStep()
	step.Id = deployId(host)
	step.Description = "deploy " + host.Name
	step.Action = deployAction
	step.OnFailure = ""

	for _, dependency := range dependencies {
		step.DependsOn = append(step.DependsOn, dependency.Id)
	}

	step.Options["host"] = host // FIXME: What is actually needed here?

	return step
}

func CreateStepReboot(host nix.Host) Step {
	step := EmptyStep()
	step.Description = "reboot " + host.Name
	step.Action = "reboot"

	step.Options["host"] = host // FIXME: What is actually needed here?

	return step
}

func CreateStepIsOnline(host nix.Host) Step {
	step := EmptyStep()
	step.Description = "test if " + host.Name + " is online"
	step.Action = "is-online"

	step.Options["host"] = host

	return step
}

func CreateStepRepeatUntilSuccess() Step {
	step := EmptyStep()
	step.Description = "repeat sub-steps until success"
	step.Action = "repeat-until-success"

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

	if step.Action == "" || step.Action == "none" {
		writer.WriteString(fmt.Sprintf("\t\"%s\"[label = \"<f0> wrapper | <f1> %s\", shape=record, color=grey64, fontcolor=grey64, style=\"rounded,dashed\"]\n", step.Id, step.Description))
	} else if step.Action == "skip" {
		writer.WriteString(fmt.Sprintf("\t\"%s\"[label = \"<f0> skipped | <f1> %s\", shape=record, color=grey64, style=\"rounded,dashed\"]\n", step.Id, step.Description))
	} else {
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
