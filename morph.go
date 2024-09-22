package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/DBCDK/kingpin"
	"github.com/DBCDK/morph/common"
	"github.com/DBCDK/morph/cruft"
	"github.com/DBCDK/morph/executors"
	"github.com/DBCDK/morph/nix"
	"github.com/DBCDK/morph/planner"
	"github.com/DBCDK/morph/ssh"
	"github.com/DBCDK/morph/utils"
)

// These are set at build time via -ldflags magic
var version string
var assetRoot string

var switchActions = []string{"dry-activate", "test", "switch", "boot"}

var (
	app                 = kingpin.New("morph", "NixOS host manager").Version(version)
	dryRun              = app.Flag("dry-run", "Don't do anything, just eval and print changes").Default("False").Bool()
	selectGlob          string
	selectTags          string
	selectEvery         int
	selectSkip          int
	selectLimit         int
	orderingTags        string
	deployment          string
	timeout             int
	askForSudoPasswd    bool
	passCmd             string
	nixBuildArg         []string
	nixBuildTarget      string
	nixBuildTargetFile  string
	build               = buildCmd(app.Command("build", "Evaluate and build deployment configuration to the local Nix store"))
	eval                = evalCmd(app.Command("eval", "Inspect value of an attribute without building"))
	push                = pushCmd(app.Command("push", "Build and transfer items from the local Nix store to target machines"))
	deploy              = deployCmd(app.Command("deploy", "Build, push and activate new configuration on machines according to switch-action"))
	deploySwitchAction  string
	deployUploadSecrets bool
	deployReboot        bool
	skipHealthChecks    bool
	skipPreDeployChecks bool
	showTrace           bool
	healthCheck         = healthCheckCmd(app.Command("check-health", "Run health checks"))
	uploadSecrets       = uploadSecretsCmd(app.Command("upload-secrets", "Upload secrets"))
	listSecrets         = listSecretsCmd(app.Command("list-secrets", "List secrets"))
	asJson              bool
	attrkey             string
	execute             = executeCmd(app.Command("exec", "Execute arbitrary commands on machines"))
	executeCommand      []string
	keepGCRoot          = app.Flag("keep-result", "Keep latest build in .gcroots to prevent it from being garbage collected").Default("False").Bool()
	allowBuildShell     = app.Flag("allow-build-shell", "Allow using `network.buildShell` to build in a nix-shell which can execute arbitrary commands on the local system").Default("False").Bool()
	planOnly            = app.Flag("plan-only", "Print the execution plan and exit").Default("False").Bool()
	hostsMap            = make(map[string]nix.Host)
	dotFile             = app.Flag("dot-file", "file to write plan to as a Graphviz dot-file").String()
)

func deploymentArg(cmd *kingpin.CmdClause) {
	cmd.Arg("deployment", "File containing the nix deployment expression").
		HintFiles("nix").
		Required().
		ExistingFileVar(&deployment)
}

func attributeArg(cmd *kingpin.CmdClause) {
	cmd.Arg("attribute", "Name of attribute to inspect").
		Required().
		StringVar(&attrkey)
}

func timeoutFlag(cmd *kingpin.CmdClause) {
	cmd.Flag("timeout", "Seconds to wait for commands/healthchecks on a host to complete").
		Default("0").
		IntVar(&timeout)
}

func askForSudoPasswdFlag(cmd *kingpin.CmdClause) {
	cmd.
		Flag("passwd", "Whether to ask interactively for remote sudo password when needed").
		Default("False").
		BoolVar(&askForSudoPasswd)
}

func getSudoPasswdCommand(cmd *kingpin.CmdClause) {
	cmd.
		Flag("passcmd", "Specify command to run for sudo password").
		Default("").
		StringVar(&passCmd)
}

func selectorFlags(cmd *kingpin.CmdClause) {
	cmd.Flag("on", "Glob for selecting servers in the deployment").
		Default("*").
		StringVar(&selectGlob)
	cmd.Flag("tagged", "Select hosts with these tags").
		Default("").
		StringVar(&selectTags)
	cmd.Flag("every", "Select every n hosts").
		Default("1").
		IntVar(&selectEvery)
	cmd.Flag("skip", "Skip first n hosts").
		Default("0").
		IntVar(&selectSkip)
	cmd.Flag("limit", "Select at most n hosts").
		IntVar(&selectLimit)
	cmd.Flag("order-by-tags", "Order hosts by tags (comma separated list)").
		Default("").
		StringVar(&orderingTags)
}

func nixBuildArgFlag(cmd *kingpin.CmdClause) {
	cmd.Flag("build-arg", "Extra argument to pass on to nix-build command. **DEPRECATED**").
		StringsVar(&nixBuildArg)
}

func nixBuildTargetFlag(cmd *kingpin.CmdClause) {
	cmd.Flag("target", "A Nix lambda defining the build target to use instead of the default").
		StringVar(&nixBuildTarget)
}

func nixBuildTargetFileFlag(cmd *kingpin.CmdClause) {
	cmd.Flag("target-file", "File containing a Nix attribute set, defining build targets to use instead of the default").
		HintFiles("nix").
		ExistingFileVar(&nixBuildTargetFile)
}

func skipHealthChecksFlag(cmd *kingpin.CmdClause) {
	cmd.
		Flag("skip-health-checks", "Whether to skip all health checks").
		Default("False").
		BoolVar(&skipHealthChecks)
}

func skipPreDeployChecksFlag(cmd *kingpin.CmdClause) {
	cmd.
		Flag("skip-pre-deploy-checks", "Whether to skip all pre-deploy checks").
		Default("False").
		BoolVar(&skipPreDeployChecks)
}

func showTraceFlag(cmd *kingpin.CmdClause) {
	cmd.
		Flag("show-trace", "Whether to pass --show-trace to all nix commands").
		Default("False").
		BoolVar(&showTrace)
}

func asJsonFlag(cmd *kingpin.CmdClause) {
	cmd.
		Flag("json", "Whether to format the output as JSON instead of plaintext").
		Default("False").
		BoolVar(&asJson)
}

func evalCmd(cmd *kingpin.CmdClause) *kingpin.CmdClause {
	deploymentArg(cmd)
	attributeArg(cmd)
	return cmd
}

func buildCmd(cmd *kingpin.CmdClause) *kingpin.CmdClause {
	selectorFlags(cmd)
	showTraceFlag(cmd)
	nixBuildArgFlag(cmd)
	nixBuildTargetFlag(cmd)
	nixBuildTargetFileFlag(cmd)
	deploymentArg(cmd)
	return cmd
}

func pushCmd(cmd *kingpin.CmdClause) *kingpin.CmdClause {
	selectorFlags(cmd)
	showTraceFlag(cmd)
	deploymentArg(cmd)
	return cmd
}

func executeCmd(cmd *kingpin.CmdClause) *kingpin.CmdClause {
	selectorFlags(cmd)
	showTraceFlag(cmd)
	askForSudoPasswdFlag(cmd)
	getSudoPasswdCommand(cmd)
	timeoutFlag(cmd)
	deploymentArg(cmd)
	cmd.
		Arg("command", "Command to execute").
		Required().
		StringsVar(&executeCommand)
	cmd.NoInterspersed = true
	return cmd
}

func deployCmd(cmd *kingpin.CmdClause) *kingpin.CmdClause {
	selectorFlags(cmd)
	showTraceFlag(cmd)
	nixBuildArgFlag(cmd)
	deploymentArg(cmd)
	timeoutFlag(cmd)
	askForSudoPasswdFlag(cmd)
	getSudoPasswdCommand(cmd)
	skipHealthChecksFlag(cmd)
	skipPreDeployChecksFlag(cmd)
	cmd.
		Flag("upload-secrets", "Upload secrets as part of the host deployment").
		Default("False").
		BoolVar(&deployUploadSecrets)
	cmd.
		Flag("reboot", "Reboots the host after system activation, but before healthchecks has executed.").
		Default("False").
		BoolVar(&deployReboot)
	cmd.
		Arg("switch-action", "Either of "+strings.Join(switchActions, "|")).
		Required().
		HintOptions(switchActions...).
		EnumVar(&deploySwitchAction, switchActions...)
	return cmd
}

func healthCheckCmd(cmd *kingpin.CmdClause) *kingpin.CmdClause {
	selectorFlags(cmd)
	showTraceFlag(cmd)
	deploymentArg(cmd)
	timeoutFlag(cmd)
	return cmd
}

func uploadSecretsCmd(cmd *kingpin.CmdClause) *kingpin.CmdClause {
	selectorFlags(cmd)
	showTraceFlag(cmd)
	askForSudoPasswdFlag(cmd)
	getSudoPasswdCommand(cmd)
	skipHealthChecksFlag(cmd)
	deploymentArg(cmd)
	return cmd
}

func listSecretsCmd(cmd *kingpin.CmdClause) *kingpin.CmdClause {
	selectorFlags(cmd)
	showTraceFlag(cmd)
	deploymentArg(cmd)
	asJsonFlag(cmd)
	return cmd
}

func setup() {
	utils.ValidateEnvironment("nix")

	utils.SignalHandler()

	if assetRoot == "" {
		common.HandleError(errors.New("Morph must be compiled with \"-ldflags=-X main.assetRoot=<path-to-installed-data/>\"."))
	}
}

func main() {

	clause := kingpin.MustParse(app.Parse(os.Args[1:]))

	//TODO: Remove deprecation warning when removing --build-arg flag
	if len(nixBuildArg) > 0 {
		fmt.Fprintln(os.Stderr, "Deprecation: The --build-arg flag will be removed in a future release.")
	}

	defer utils.RunFinalizers()
	setup()

	mctx := &common.MorphContext{
		SSHContext:          ssh.CreateSSHContext(askForSudoPasswd, passCmd),
		NixContext:          nix.GetNixContext(assetRoot, showTrace, *keepGCRoot, *allowBuildShell),
		AssetRoot:           assetRoot,
		AttrKey:             attrkey,
		Deployment:          deployment,
		DeploySwitchAction:  deploySwitchAction,
		DeployReboot:        deployReboot,
		DeployUploadSecrets: deployUploadSecrets,
		DryRun:              *dryRun,
		ExecuteCommand:      executeCommand,
		NixBuildArg:         nixBuildArg,
		NixBuildTarget:      nixBuildTarget,
		NixBuildTargetFile:  nixBuildTargetFile,
		OrderingTags:        orderingTags, // FIXME: should these be split already here?
		SelectEvery:         selectEvery,
		SelectGlob:          selectGlob,
		SelectLimit:         selectLimit,
		SelectSkip:          selectSkip,
		SelectTags:          selectTags, // FIXME: should these be split already here?
		SkipHealthChecks:    skipHealthChecks,
		SkipPreDeployChecks: skipPreDeployChecks,
		Timeout:             timeout,
	}

	// evaluate without building hosts
	switch clause {
	case eval.FullCommand():
		_, err := cruft.ExecEval(mctx)
		common.HandleError(err)
		return
	}

	// setup hosts
	hosts, err := cruft.GetHosts(mctx, deployment)
	common.HandleError(err)

	plan := createPlan(hosts, clause)

	planJson, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(planJson))

	if dotFile != nil && *dotFile != "" {
		f, err := os.Create(*dotFile)
		if err != nil {
			panic(err)
		}
		defer f.Close()

		writer := bufio.NewWriter(f)
		planner.WriteDotFile(writer, plan)
		writer.Flush()
	}

	if *planOnly { // FIXME: Is this really *dryRun instead? Not sure
		// Don't execute the plan
		return
	}

	executor := executors.DefaultPlanExecutor{
		Hosts:        hostsMap,
		MorphContext: mctx,
		SSHContext:   ssh.CreateSSHContext(askForSudoPasswd, passCmd),
		NixContext:   nix.GetNixContext(assetRoot, showTrace, *keepGCRoot, *allowBuildShell),
	}

	err = executor.Init()
	if err != nil {
		panic(err)
	}

	err = planner.ExecutePlan(executor, plan)
	if err != nil {
		panic(err)
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

func createPlan(hosts []nix.Host, clause string) planner.Step {
	hostSpecificPlans := make(map[string]planner.Step, 0)

	for _, host := range hosts {
		hostsMap[host.Name] = host

		hostSpecificPlan := planner.EmptyStep()
		hostSpecificPlan.Description = "host: " + host.Name
		hostSpecificPlan.Parallel = false

		hostSpecificPlans[host.Name] = hostSpecificPlan
	}

	fmt.Println("Execution plan:")

	plan := planner.EmptyStep()
	plan.Description = "Root of execution plan"

	buildPlan := planner.CreateBuildPlan(hosts)

	stepGetSudoPasswd := planner.CreateStepGetSudoPasswd()

	if askForSudoPasswd {
		plan = planner.AddSteps(plan, stepGetSudoPasswd)
	}

	switch clause {
	case build.FullCommand():

		plan = planner.AddSteps(plan, buildPlan)

	case push.FullCommand():

		plan = planner.AddSteps(plan, buildPlan)

		for _, host := range hosts {
			push := planner.CreateStepPush(host)
			push.DependsOn = append(push.DependsOn, buildPlan.Id)

			hostSpecificPlans[host.Name] = planner.AddStepsSeq(
				hostSpecificPlans[host.Name],
				push,
			)
		}

	case deploy.FullCommand():
		fmt.Println("Execution plan: deploy: Not implemented")

		plan = planner.AddSteps(plan, buildPlan)

		for _, host := range hosts {
			push := planner.CreateStepPush(host)
			push.DependsOn = append(push.DependsOn, buildPlan.Id)

			deployDryActivate := planner.CreateStepDeployDryActivate(host)
			deploySwitch := planner.CreateStepDeploySwitch(host)
			deployTest := planner.CreateStepDeployTest(host)
			deployBoot := planner.CreateStepDeployBoot(host)

			stepReboot := planner.CreateStepReboot(host)

			stepWaitForOnline := planner.CreateStepWaitForOnline(host)

			preDeployChecks := planner.CreateStepChecks(
				host,
				make([]planner.CommandPlus, 0),
				planner.HealthChecksToCommands(host.PreDeployChecks.Cmd),
				make([]planner.RequestPlus, 0),
				planner.HealthChecksToRequests(host.PreDeployChecks.Http),
			)

			healthChecks := planner.CreateStepChecks(
				host,
				make([]planner.CommandPlus, 0),
				planner.HealthChecksToCommands(host.HealthChecks.Cmd),
				make([]planner.RequestPlus, 0),
				planner.HealthChecksToRequests(host.HealthChecks.Http),
			)

			if skipPreDeployChecks {
				preDeployChecks = planner.CreateStepSkip(preDeployChecks)
			}

			if skipHealthChecks {
				healthChecks = planner.CreateStepSkip(healthChecks)
			}

			if askForSudoPasswd {
				deployDryActivate.DependsOn = append(deployDryActivate.DependsOn, stepGetSudoPasswd.Id)
				deploySwitch.DependsOn = append(deploySwitch.DependsOn, stepGetSudoPasswd.Id)
				deployTest.DependsOn = append(deployTest.DependsOn, stepGetSudoPasswd.Id)
				deployBoot.DependsOn = append(deployBoot.DependsOn, stepGetSudoPasswd.Id)
				stepReboot.DependsOn = append(stepReboot.DependsOn, stepGetSudoPasswd.Id)
			}

			switch deploySwitchAction {
			case "dry-activate":
				hostSpecificPlans[host.Name] = planner.AddStepsSeq(
					hostSpecificPlans[host.Name],
					push,
					deployDryActivate,
				)

			case "test":
				// FIXME: requires upload secrets
				hostSpecificPlans[host.Name] = planner.AddStepsSeq(
					hostSpecificPlans[host.Name],
					push,
					preDeployChecks,
					deployTest,
					healthChecks,
				)

			case "switch":
				hostSpecificPlans[host.Name] = planner.AddStepsSeq(
					hostSpecificPlans[host.Name],
					push,
					preDeployChecks,
					deploySwitch,
					healthChecks,
				)

			case "boot":
				// FIXME: requires upload secrets
				hostSpecificPlans[host.Name] = planner.AddStepsSeq(
					hostSpecificPlans[host.Name],
					push,
					preDeployChecks,
					deployBoot,
				)
			}

			// reboot can be added to any action, even if weird..
			if deployReboot {
				hostSpecificPlans[host.Name] = planner.AddStepsSeq(
					hostSpecificPlans[host.Name],
					stepReboot,
					stepWaitForOnline,
					healthChecks,
				)
			}

		}

	case healthCheck.FullCommand():

		plan = planner.AddSteps(plan, buildPlan)

		for _, host := range hosts {
			push := planner.CreateStepPush(host)
			push.DependsOn = append(push.DependsOn, buildPlan.Id)

			healthChecks := planner.CreateStepChecks(
				host,
				make([]planner.CommandPlus, 0),
				planner.HealthChecksToCommands(host.HealthChecks.Cmd),
				make([]planner.RequestPlus, 0),
				planner.HealthChecksToRequests(host.HealthChecks.Http),
			)

			hostSpecificPlans[host.Name] = planner.AddStepsSeq(
				hostSpecificPlans[host.Name],
				push,
				healthChecks,
			)
		}

	case uploadSecrets.FullCommand():
		fmt.Println("Execution plan: upload-secrets: Not implemented")

	case listSecrets.FullCommand():
		fmt.Println("Execution plan: list-secrets: Not implemented")

	case execute.FullCommand():
		fmt.Println("Execution plan: execute: Not implemented")

	}

	for _, serverPlan := range hostSpecificPlans {
		// FIXME: This is a bit too much of a hack just to
		// avoid build including empty plans
		if len(serverPlan.Steps) > 0 {
			plan = planner.AddSteps(plan, serverPlan)
		}
	}

	return plan
}
