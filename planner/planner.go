package planner

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/DBCDK/morph/common"
	"github.com/DBCDK/morph/nix"
	"github.com/DBCDK/morph/ssh"
)

var (
	stepsDone     = make(map[string]*sync.WaitGroup)
	stepsDoneChan = make(chan StepStatus)
)

type PlanExecutor interface {
	Init() error
	TearDown() error
	GetMorphContext() *common.MorphContext // FIXME: Get rid of this or limit it a lot
	GetSSHContext() *ssh.SSHContext
	GetNixContext() *nix.NixContext

	GetHosts() map[string]nix.Host

	Build(step Step) error
	Push(step Step) error
	DeploySwitch(step Step) error
	DeployBoot(step Step) error
	DeployDryActivate(step Step) error
	DeployTest(step Step) error
	Reboot(step Step) error
	CommandCheckLocal(step Step) error
	CommandCheckRemote(step Step) error
	HttpCheckLocal(step Step) error
	HttpCheckRemote(step Step) error
}

func ExecutePlan(executor PlanExecutor, plan Step) error {
	// THese should be started somewhere better
	go plannerStepStatusWriter()

	return ExecuteStep(executor, plan)
}

func ExecuteStep(executor PlanExecutor, step Step) error {
	fmt.Printf("Running step %s: %s (dependencies: %v)\n", step.Action, step.Description, step.DependsOn)

	stepsDoneChan <- StepStatus{Id: step.Id, Status: "started"}

	waitForDependencies(step.Id, "dependencies", step.DependsOn)

	switch step.Action {
	case "build":
		executor.Build(step)

	case "push":
		executor.Push(step)

	case "none":
		fallthrough
	case "skip":
		fallthrough
	case "":
		// wrapper step, nothing to do
	}

	if step.Parallel {
		var wg sync.WaitGroup

		for _, subStep := range step.Steps {
			wg.Add(1)
			go func(step Step) {
				defer wg.Done()
				ExecuteStep(executor, step)
			}(subStep)
		}

		// Wait for children to all be ready, before making the current step ready
		wg.Wait()

	} else {
		for _, subStep := range step.Steps {
			ExecuteStep(executor, subStep)
		}
	}

	stepsDoneChan <- StepStatus{Id: step.Id, Status: "done"}

	return nil
}

func plannerStepStatusWriter() {
	for stepStatus := range stepsDoneChan {
		fmt.Printf("step update: %s = %s\n", stepStatus.Id, stepStatus.Status)
		switch strings.ToLower(stepStatus.Status) {
		case "started":
			var stepWg = &sync.WaitGroup{}
			stepWg.Add(1)

			stepsDone[stepStatus.Id] = stepWg
		case "done":
			stepsDone[stepStatus.Id].Done()

		default:
			panic("Only status=started and status=done allowed")
		}
	}
}

func waitForDependencies(id string, hint string, dependencies []string) {
	fmt.Printf("%s: depends on %d steps: %v\n", id, len(dependencies), dependencies)

	for _, dependency := range dependencies {
		for {
			// Wait for the dependency to actually start running
			// It's probably better to pre-create all dependencies so this isn't necessary
			fmt.Printf("%s: %s: waiting for %s to start\n", id, hint, dependency)

			if dependencyWg, dependencyStarted := stepsDone[dependency]; dependencyStarted {
				// Wait for dependency to finish runningCheck if the dependencies update channel has been closed (=> it's done), and break the loop
				dependencyWg.Wait()
				fmt.Printf("%s: %s: %s done\n", id, hint, dependency)
				break

			}

			// Sleep if we haven't seen the dependency
			time.Sleep(1 * time.Second)
		}
	}
}
