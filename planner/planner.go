package planner

import (
	"context"
	"errors"
	"fmt"
	"github.com/DBCDK/morph/cache"
	"github.com/DBCDK/morph/common"
	"github.com/DBCDK/morph/nix"
	"github.com/DBCDK/morph/ssh"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"sort"
	"time"

	"golang.org/x/sync/errgroup"
)

const (
	Scheduled string = "scheduled"
	Running   string = "running"
	Done      string = "done"
)

// FIXME: IDEA: Deployment simulation - make a fake MorphContext where things like SSH-calls are faked and logged instead

type MegaContext struct { // FIXME: Lol get rid of this
	Hosts        map[string]nix.Host
	MorphContext *common.MorphContext
	SSHContext   *ssh.SSHContext
	NixContext   *nix.NixContext
	Cache        *cache.LockedMap[string]
	StepsDone    *cache.LockedMap[string]
	Steps        *cache.LockedMap[Step]
	//State tilføj steps and bla bla, stat
}

type ExecutionState struct {
	Steps     *cache.LockedMap[Step]
	StepsDone *cache.LockedMap[string]
}

func (mega *MegaContext) UpdateStepStatus(stepId string, status string) {
	log.Info().
		Str("event", "step-status").
		Str("step", stepId).
		Str("status", status).
		Msg("step update")

	mega.StepsDone.Update(stepId, status)
}

func StepMonitor(stepsDb *cache.LockedMap[Step], m *cache.LockedMap[string]) {
	for {
		data := m.GetCopy()

		stepIds := make([]string, 0)
		for stepId, _ := range data {
			stepIds = append(stepIds, stepId)
		}

		sort.Strings(stepIds)

		for _, stepId := range stepIds {
			step, _ := stepsDb.Get(stepId)
			log.Debug().
				Dict("step", zerolog.Dict().
					Str("id", step.Id).
					Str("action", step.ActionName)).
				Msg(fmt.Sprintf("step: " + stepId + " state: " + data[stepId]))
		}

		time.Sleep(time.Second)
	}
}

func ExecuteStep(ctx context.Context, megaCtx MegaContext, step Step) error {
	megaCtx.Steps.Update(step.Id, step)

	megaCtx.UpdateStepStatus(step.Id, Scheduled)
	waitForDependencies(megaCtx, step.Id, "dependencies", step.DependsOn)
	megaCtx.UpdateStepStatus(step.Id, Running)

	err := step.Action.Run(ctx, megaCtx.MorphContext, megaCtx.Hosts, megaCtx.Cache)
	if err != nil {
		return err
	}

	if step.Parallel {
		group, ctx := errgroup.WithContext(ctx)

		for _, subStep := range step.Steps {
			step := subStep

			switch step.OnFailure {
			case "retry":
				// repeat until success
				group.Go(func() error {
					err := errors.New("fake error")

					for err != nil {
						err = ExecuteStep(ctx, megaCtx, step)
						if err != nil {
							log.Error().Err(err).Msg("Error while running step (will retry)")
						}
					}

					return nil
				})

			case "ignore":
				// ok no matter what
				group.Go(func() error {
					err := ExecuteStep(ctx, megaCtx, step)
					if err != nil {
						log.Error().Err(err).Msg("Error while running step (ignored)")
					}

					return nil
				})

			default:
				// return err on err

				group.Go(func() error {
					return ExecuteStep(ctx, megaCtx, step)
				})
			}
		}

		// Wait for children to all be ready, before making the current step ready
		if err := group.Wait(); err != nil {
			return err
		}

	} else {
		for _, subStep := range step.Steps {
			err := ExecuteStep(ctx, megaCtx, subStep)
			if err != nil {
				return err
			}
		}
	}

	megaCtx.UpdateStepStatus(step.Id, Done)

	return nil
}

func waitForDependencies(megaContext MegaContext, id string, hint string, dependencies []string) {
	if len(dependencies) == 0 {
		return
	}

	dependenciesStillWaiting := make([]string, 0)

	for _, dependency := range dependencies {
		dependenciesStillWaiting = append(dependenciesStillWaiting, dependency)
	}

	fmt.Printf("%s: depends on %d steps: %v\n", id, len(dependencies), dependencies)

	for {
		if len(dependenciesStillWaiting) == 0 {
			break
		}

		zLogWaitingDependencies := zerolog.Arr()

		for _, dependency := range dependenciesStillWaiting {
			zLogWaitingDependencies.Str(dependency)
		}

		log.Info().
			Str("event", "step-blocked").
			Str("step", id).
			Array("blocked-by", zLogWaitingDependencies).
			Msg("step blocked by dependencies")

		dependenciesStillWaiting = make([]string, 0)

		for _, dependency := range dependencies { // FIXME: optimize by only looking at dependenciesStillWaiting
			status, err := megaContext.StepsDone.Get(dependency)
			if err != nil {
				// Not started
				dependenciesStillWaiting = append(dependenciesStillWaiting, dependency)
			} else {
				if status != Done {
					dependenciesStillWaiting = append(dependenciesStillWaiting, dependency)
				}
			}
		}

		time.Sleep(1 * time.Second)
	}
}
