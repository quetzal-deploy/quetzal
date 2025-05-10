package main

// TODO: Quetzal NixOS integration tests
// TODO: turn all `panic`'s into proper error handling
// TODO: remove --passwd since Quetzal can then ignore stdin and doesn't have to figure out how to share it between steps
// TODO: 12:14AM ERR error marshalling plan to JSON error="json: error calling MarshalJSON for type steps.Step: json: error calling MarshalJSON for type steps.Step: json: error calling MarshalJSON for type steps.Step: unmarshall: unknown action: wait-for-online"
//     ^ drop wait-for-online and let it be a repeating gate/step kinda thing that calls IsOnline
import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/kr/pretty"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/DBCDK/kingpin"

	"github.com/quetzal-deploy/quetzal/cliparser"
	"github.com/quetzal-deploy/quetzal/common"
	"github.com/quetzal-deploy/quetzal/cruft"
	"github.com/quetzal-deploy/quetzal/events"
	"github.com/quetzal-deploy/quetzal/internal/constraints"
	"github.com/quetzal-deploy/quetzal/internal/daemon"
	"github.com/quetzal-deploy/quetzal/internal/http"
	"github.com/quetzal-deploy/quetzal/nix"
	"github.com/quetzal-deploy/quetzal/planner"
	"github.com/quetzal-deploy/quetzal/steps"
	"github.com/quetzal-deploy/quetzal/ui"
	"github.com/quetzal-deploy/quetzal/utils"
)

// These are set at build time via -ldflags magic
var version string
var assetRoot string

func setup() {
	utils.ValidateEnvironment("nix")

	utils.SignalHandler()

	if assetRoot == "" {
		common.HandleError(errors.New("Quetzal must be compiled with \"-ldflags=-X main.assetRoot=<path-to-installed-data/>\"."))
	}
}

func main() {

	// TODO: Implement "context" part of a plan is running on:
	// Default context = local
	// A context can change the host something is running on
	// nix copy Quetzal itself to the new host, and execute it with the sub-plan
	// eg context:local -> repeat-until-success -> context:$host -> exec command (health check)
	// Running the deploy from Matrix can also be a context, and the local Quetzal will wait for it to finish
	// Try substitute on remote host before building locally (to avoid downloaded things that will be substituted from cache
	// Wrap exec.Command to unify logging the command and to give each command a unique ID that can be used to reconstruct what is being logged
	// Find a way to log commands where host is the machine executing Quetzal
	// Detect when a process wants STDIN-input and find a way to handle that - mostly relevant for SSH TOFU prompts
	// Tyvstjæl hvordan K8s deployments virker med min og max men brug det på tags. Når dette aktiveres skal Quetzal først health checke alt så Quetzal ved hvor meget der er oppe og nede
	// Replace tags with labels
	// Canary mode: Ramp up concurrency along with successful updates
	// Some steps should be implicit, like running nix-build on demand
	// Poweroff and poweron actions, maybe something like maintenancemode, or... even custom defined host states
	// SSH output needs to be log'ed instead

	// Constraints must be serialized into the plan (and maybe that makes us actually need a Plan type that wraps the first Step because of that). Otherwise constraints are lost when dumping and loading a plan.

	// Metrics that can be reacted on

	cli, cmdClauses, opts := cliparser.New(version, assetRoot)

	clause := kingpin.MustParse(cli.Parse(os.Args[1:]))

	if cmdClauses.Daemon.FullCommand() == clause {
		// force JSON-output (and thus no UI)
		*opts.JsonOut = true
	}

	eventManager := events.NewManager()
	logManager := events.NewManager()

	go func() {
		for event := range eventManager.Subscribe() {
			switch event.Event.(type) {
			case events.QueueStatus:

			default:
				log.Debug().Msg("")
				log.Debug().Msg(pretty.Sprint(event.Event))
			}
		}
	}()

	// Don't actually run the UI unless activated
	tui := ui.DoTea(eventManager)

	if !*opts.JsonOut {
		go func() {
			if _, err := tui.Run(); err != nil {
				fmt.Printf("Quetzal failed: %v", err)
				os.Exit(1)
			}
		}()

		log.Logger = log.Output(zerolog.ConsoleWriter{
			Out:     logManager.NewLogWriter(),
			NoColor: true,
		})

	} else {
		//log.Logger = log.Output(zerolog.New(os.Stdout).With().Timestamp().Logger())
		log.Logger = log.Output(os.Stdout)

		// FIXME: Output events to stdout, output log.bla to stderr
	}

	defer utils.RunFinalizers()
	setup()

	// evaluate without building hosts
	switch clause {
	case cmdClauses.Eval.FullCommand():
		_, err := cruft.ExecEval(opts)
		common.HandleError(err)
		return
	}

	switch clause {

	case cmdClauses.Daemon.FullCommand():
		daemon := daemon.NewDaemon(opts)
		daemon.LoadDeployments(opts.DeploymentsDir)
		http.Run(&daemon, 8123, eventManager, opts.DeploymentsDir)
		return

	case cmdClauses.PlanRun.FullCommand():
		// FIXME: embed plan instead of file path
		log.Info().
			Str("plan", opts.PlanFile).
			Msg("running plan")

	case cmdClauses.PlanResume.FullCommand():
		// FIXME: embed plan instead of file path
		log.Info().
			Str("plan", opts.PlanFile).
			Msg("resuming plan")

	default:
		// setup hosts
		// FIXME: Should this be its own step instead?
		// But then what about the generated plan? It can no longer contain the lists of hosts, but only the deployment and filters users, which will make resume difficult..
		deploymentMetadata, hosts, err := cruft.GetHosts(opts)
		common.HandleError(err)

		hostsMap := make(map[string]nix.Host)

		for _, host := range hosts {
			hostsMap[host.Name] = host
		}

		plan := createPlan(cmdClauses, opts, hosts, clause)
		if !*opts.JsonOut {
			eventManager.SendEvent(events.RegisterPlan{Plan: plan})
		}

		planJson, err := json.Marshal(plan)
		if err != nil {
			log.Error().Err(err).Msg("error marshalling plan to JSON")
			return
		}

		log.Info().
			Str("event", "plan").
			RawJSON("plan", planJson).
			Msg("Generated plan")

		if opts.DotFile != nil && *opts.DotFile != "" {
			f, err := os.Create(*opts.DotFile)
			if err != nil {
				panic(err)
			}
			defer f.Close()

			writer := bufio.NewWriter(f)
			planner.WriteDotFile(writer, plan)
			writer.Flush()
		}

		if *opts.PlanOnly { // FIXME: Is this really *dryRun instead? Not sure
			// Don't execute the plan
			return
		}

		mergedConstraints := make([]constraints.Constraint, 0)
		defaultConstraints := make([]constraints.Constraint, 0)

		constraintsArgs, err := cliparser.ParseConstraints(*opts.ConstraintsFlag)
		if err != nil {
			log.Fatal().Err(err).Msg(err.Error())
		}

		mergedConstraints = append(mergedConstraints, constraintsArgs...)
		mergedConstraints = append(mergedConstraints, deploymentMetadata.Constraints...)

		defaultConstraints = append(defaultConstraints, constraints.NewConstraint(constraints.LabelSelector{Label: "_", Value: "host"}, 1))

		for _, defaultConstraint := range defaultConstraints {
			add := true

			for _, constraint := range mergedConstraints {
				if defaultConstraint.Selector.Match(constraint.Selector.Label, constraint.Selector.Value) {
					add = false
					break
				}
			}

			if add {
				mergedConstraints = append(mergedConstraints, defaultConstraint)
			}
		}

		log.Debug().Msg("constraints:")
		for _, c := range mergedConstraints {
			log.Debug().Msg(fmt.Sprintf("- %s=%s: %v\n", c.Selector.Label, c.Selector.Value, c))
		}
		eventManager.SendEvent(events.Debug{Data: fmt.Sprintf("constraints: %#v", mergedConstraints)})

		if false {
			os.Exit(17)
		}

		planner_ := planner.NewPlanner(eventManager, hostsMap, opts, mergedConstraints)

		go planner_.StepMonitor()

		planner_.QueueStep(plan)

		go http.Run(nil, 8123, eventManager, opts.DeploymentsDir)

		err = planner_.Run(context.TODO())
		if err != nil {
			log.Error().Err(err).Msg("Error while running step") // FIXME: Log the offending step/action somehow
			// FIXME: Dump the plan with status on what was done, and what wasn't, so it can be resumed
			if *opts.JsonOut {
				// don't os.Exit if running with UI
				os.Exit(1)
			}
		}

		if !*opts.JsonOut {
			// Let the user terminate Quetzal from the UI
			tui.Wait()
		}
	}

	// switch clause {
	// case build.FullCommand():
	// 	_, err = execBuild(hosts)
	// case push.FullCommand():
	// 	_, err = execPush(hosts)
	// case deploy.FullCommand():
	// 	_, err = execDeploy(hosts)
	// case healthCheck.FullCommand():
	// 	err = execHealthCheck(hosts)
	// case uploadSecrets.FullCommand():
	// 	err = execUploadSecrets(createSSHContext(), hosts, nil)
	// case listSecrets.FullCommand():
	// 	if asJson {
	// 		err = execListSecretsAsJson(hosts)
	// 	} else {
	// 		execListSecrets(hosts)
	// 	}
	// case execute.FullCommand():
	// 	err = execExecute(hosts)
	// }

	// common.HandleError(err)
}

// TODO: Different planners should have default constraints exposed, to be displayed in the UI for suggestions to override
// FIXME: Refactor to not need cmdClauses and opts
func createPlan(cmdClauses *cliparser.KingpinCmdClauses, opts *common.QuetzalOptions, hosts []nix.Host, clause string) steps.Step {
	plan := steps.New().
		Id("root").
		Description("Root of execution plan").
		Parallel()

	buildPlan := planner.CreateBuildPlan(hosts)

	hostSpecificPlans := make(map[string]*steps.StepBuilder, 0)

	for _, host := range hosts {

		hostSpecificPlan := steps.New().
			Id("host:"+host.Name).
			Description("host: "+host.Name).
			Labels(host.Labels).
			Label("_", "host").
			Action(&steps.None{}).
			AddDependencies(buildPlan.Id)

		if _, hasHostLabel := host.Labels["host"]; !hasHostLabel {
			hostSpecificPlan.Label("host", host.Name) // TODO: Document implicit labels
		}

		hostSpecificPlans[host.Name] = hostSpecificPlan
	}

	stepGetSudoPasswd := planner.CreateStepGetSudoPasswd()

	if opts.AskForSudoPasswd {
		plan.AddSteps(stepGetSudoPasswd)
	}

	switch clause {
	case cmdClauses.Build.FullCommand():

		plan.AddSteps(buildPlan)

	case cmdClauses.Push.FullCommand():

		plan.AddSteps(buildPlan)

		for _, host := range hosts {
			push := planner.CreateStepPush(host)

			hostSpecificPlans[host.Name].AddSequentialSteps(push)
		}

	case cmdClauses.Deploy.FullCommand():
		plan.AddSteps(buildPlan)

		for _, host := range hosts {
			push := planner.CreateStepPush(host)

			deployDryActivate := planner.CreateStepDeployDryActivate(host)
			deploySwitch := planner.CreateStepDeploySwitch(host)
			deployTest := planner.CreateStepDeployTest(host)
			deployBoot := planner.CreateStepDeployBoot(host)

			stepReboot := planner.CreateStepReboot(host)

			stepWaitForOnline := planner.CreateStepWaitForOnline(host)

			preDeployChecks := planner.CreateStepChecks(
				"pre-deploy-checks",
				host,
				make([]planner.CommandPlus, 0),
				planner.HealthChecksToCommands(host.PreDeployChecks.Cmd),
				make([]planner.RequestPlus, 0),
				planner.HealthChecksToRequests(host.PreDeployChecks.Http),
			)

			healthChecks := planner.CreateStepChecks(
				"healthchecks",
				host,
				make([]planner.CommandPlus, 0),
				planner.HealthChecksToCommands(host.HealthChecks.Cmd),
				make([]planner.RequestPlus, 0),
				planner.HealthChecksToRequests(host.HealthChecks.Http),
			)

			if opts.SkipPreDeployChecks {
				preDeployChecks = planner.CreateStepSkip(preDeployChecks)
			}

			if opts.SkipHealthChecks {
				healthChecks = planner.CreateStepSkip(healthChecks)
			}

			if opts.AskForSudoPasswd {
				deployDryActivate.DependsOn = append(deployDryActivate.DependsOn, stepGetSudoPasswd.Id)
				deploySwitch.DependsOn = append(deploySwitch.DependsOn, stepGetSudoPasswd.Id)
				deployTest.DependsOn = append(deployTest.DependsOn, stepGetSudoPasswd.Id)
				deployBoot.DependsOn = append(deployBoot.DependsOn, stepGetSudoPasswd.Id)
				stepReboot.DependsOn = append(stepReboot.DependsOn, stepGetSudoPasswd.Id)
			}

			switch opts.DeploySwitchAction {
			case "dry-activate":
				hostSpecificPlans[host.Name].AddSequentialSteps(
					push,
					deployDryActivate,
				)

			case "test":
				// FIXME: requires upload secrets
				hostSpecificPlans[host.Name].AddSequentialSteps(
					push,
					preDeployChecks,
					deployTest,
					healthChecks,
				)

			case "switch":
				hostSpecificPlans[host.Name].AddSequentialSteps(
					push,
					preDeployChecks,
					deploySwitch,
					healthChecks,
				)

			case "boot":
				// FIXME: requires upload secrets
				hostSpecificPlans[host.Name].AddSequentialSteps(
					push,
					deployBoot,
				)
			}

			// reboot can be added to any action, even if weird..
			if opts.DeployReboot {
				hostSpecificPlans[host.Name].AddSequentialSteps(
					stepReboot,
					stepWaitForOnline,
					healthChecks,
				)
			}
		}

	case cmdClauses.HealthCheck.FullCommand():

		plan.AddSteps(buildPlan)

		for _, host := range hosts {
			push := planner.CreateStepPush(host)

			healthChecks := planner.CreateStepChecks(
				"healthchecks",
				host,
				make([]planner.CommandPlus, 0),
				planner.HealthChecksToCommands(host.HealthChecks.Cmd),
				make([]planner.RequestPlus, 0),
				planner.HealthChecksToRequests(host.HealthChecks.Http),
			)

			hostSpecificPlans[host.Name].AddSequentialSteps(
				push,
				healthChecks,
			)
		}

	case cmdClauses.SecretsUpload.FullCommand():
		log.Error().Msg("Execution plan: deploy: Not implemented")

	case cmdClauses.SecretsList.FullCommand():
		log.Error().Msg("Execution plan: deploy: Not implemented")

	case cmdClauses.Execute.FullCommand():
		log.Error().Msg("Execution plan: execute: Not implemented")

	}

	for _, serverPlanBuilder := range hostSpecificPlans {
		serverPlan := serverPlanBuilder.Build()
		// FIXME: This is a bit too much of a hack just to
		// avoid build including empty plans
		if len(serverPlan.Steps) > 0 {
			plan.AddSteps(serverPlan)
		}
	}

	return plan.Build()
}
