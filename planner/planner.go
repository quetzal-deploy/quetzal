package planner

import (
	"fmt"
	"github.com/DBCDK/morph/cache"
	"github.com/DBCDK/morph/common"
	"github.com/DBCDK/morph/nix"
	"github.com/DBCDK/morph/ssh"
	"strings"
	"sync"
	"time"
)

var (
	stepsDone     = make(map[string]*sync.WaitGroup)
	stepsDoneChan = make(chan StepStatus)
)

// FIXME: IDEA: Deployment simulation - make a fake MorphContext where things like SSH-calls are faked and logged instead

type MegaContext struct { // FIXME: Lol get rid of this
	Hosts        map[string]nix.Host
	MorphContext *common.MorphContext
	SSHContext   *ssh.SSHContext
	NixContext   *nix.NixContext
	Cache        *cache.Cache
}

func ExecutePlan(megaCtx MegaContext, plan Step) error {
	// THese should be started somewhere better
	go plannerStepStatusWriter()

	return ExecuteStep(megaCtx, plan)
}

func ExecuteStep(megaCtx MegaContext, step Step) error {
	fmt.Printf("Running step %s: %s (dependencies: %v)\n", step.ActionName, step.Description, step.DependsOn)

	stepsDoneChan <- StepStatus{Id: step.Id, Status: "started"}

	waitForDependencies(step.Id, "dependencies", step.DependsOn)

	err := step.Action.Run(megaCtx.MorphContext, megaCtx.Hosts, megaCtx.Cache)
	if err != nil {
		return err
	}

	if step.Parallel {
		var wg sync.WaitGroup

		for _, subStep := range step.Steps {
			wg.Add(1)
			go func(step Step) {
				defer wg.Done()
				ExecuteStep(megaCtx, step)
			}(subStep)
		}

		// Wait for children to all be ready, before making the current step ready
		wg.Wait()

	} else {
		for _, subStep := range step.Steps {
			ExecuteStep(megaCtx, subStep)
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
