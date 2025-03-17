package cruft

import (
	"errors"
	"fmt"
	"github.com/DBCDK/morph/ssh"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
	"path/filepath"
	"strings"

	"github.com/DBCDK/morph/common"
	"github.com/DBCDK/morph/filter"
	"github.com/DBCDK/morph/healthchecks"
	"github.com/DBCDK/morph/nix"
	"github.com/DBCDK/morph/utils"
)

func ExecBuild(mctx *common.MorphContext, hosts []nix.Host) (string, error) {
	resultPath, err := buildHosts(mctx, hosts)
	if err != nil {
		return "", err
	}
	return resultPath, nil
}

func ExecDeploy(mctx *common.MorphContext, hosts []nix.Host) (string, error) {
	sshCtx := ssh.CreateSSHContext(mctx.Options.SshOptions())

	doPush := false
	doUploadSecrets := false
	doActivate := false

	if !*mctx.Options.DryRun {
		switch mctx.Options.DeploySwitchAction {
		case "dry-activate":
			doPush = true
			doActivate = true
		case "test":
			fallthrough
		case "switch":
			fallthrough
		case "boot":
			doPush = true
			doUploadSecrets = mctx.Options.DeployUploadSecrets
			doActivate = true
		}
	}

	resultPath, err := buildHosts(mctx, hosts)
	if err != nil {
		return "", err
	}

	fmt.Fprintln(os.Stderr)

	for _, host := range hosts {
		if host.BuildOnly {
			fmt.Fprintf(os.Stderr, "Deployment steps are disabled for build-only host: %s\n", host.Name)
			continue
		}

		singleHostInList := []nix.Host{host}

		if doPush {
			err = pushPaths(sshCtx, singleHostInList, resultPath)
			if err != nil {
				return "", err
			}
		}
		fmt.Fprintln(os.Stderr)

		if doUploadSecrets {
			phase := "pre-activation"
			err = ExecUploadSecrets(mctx, singleHostInList, &phase)
			if err != nil {
				return "", err
			}

			fmt.Fprintln(os.Stderr)
		}

		if !mctx.Options.SkipPreDeployChecks {
			err := healthchecks.PerformPreDeployChecks(sshCtx, &host, mctx.Options.Timeout)
			if err != nil {
				fmt.Fprintln(os.Stderr)
				fmt.Fprintln(os.Stderr, "Not deploying to additional hosts, since a host pre-deploy check failed.")
				utils.Exit(1)
			}
		}

		if doActivate {
			err = activateConfiguration(mctx, singleHostInList, resultPath)
			if err != nil {
				return "", err
			}
		}

		if mctx.Options.DeployReboot {
			err = host.Reboot(sshCtx)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Reboot failed")
				return "", err
			}
		}

		if doUploadSecrets {
			phase := "post-activation"
			err = ExecUploadSecrets(mctx, singleHostInList, &phase)
			if err != nil {
				return "", err
			}

			fmt.Fprintln(os.Stderr)
		}

		if !mctx.Options.SkipHealthChecks {
			err := healthchecks.PerformHealthChecks(sshCtx, &host, mctx.Options.Timeout)
			if err != nil {
				fmt.Fprintln(os.Stderr)
				fmt.Fprintln(os.Stderr, "Not deploying to additional hosts, since a host health check failed.")
				utils.Exit(1)
			}
		}

		fmt.Fprintln(os.Stderr, "Done:", host.Name)
	}

	return resultPath, nil
}

func ExecEval(mctx *common.MorphContext) (string, error) {
	deploymentFile, err := os.Open(mctx.Options.Deployment)
	deploymentPath, err := filepath.Abs(deploymentFile.Name())
	if err != nil {
		return "", err
	}

	path, err := nix.EvalHosts(mctx.NixContext, deploymentPath, mctx.Options.AttrKey)

	return path, err
}

func ExecExecute(mctx *common.MorphContext, hosts []nix.Host) error {
	sshCtx := ssh.CreateSSHContext(mctx.Options.SshOptions())

	for _, host := range hosts {
		if host.BuildOnly {
			fmt.Fprintf(os.Stderr, "Exec is disabled for build-only host: %s\n", host.Name)
			continue
		}
		fmt.Fprintln(os.Stderr, "** "+host.Name)
		sshCtx.CmdInteractive(&host, mctx.Options.Timeout, mctx.Options.ExecuteCommand...)
		fmt.Fprintln(os.Stderr)
	}

	return nil
}

func ExecHealthCheck(mctx *common.MorphContext, hosts []nix.Host) error {
	sshCtx := ssh.CreateSSHContext(mctx.Options.SshOptions())

	var err error
	for _, host := range hosts {
		if host.BuildOnly {
			fmt.Fprintf(os.Stderr, "Healthchecks are disabled for build-only host: %s\n", host.Name)
			continue
		}
		err = healthchecks.PerformHealthChecks(sshCtx, &host, mctx.Options.Timeout)
	}

	if err != nil {
		err = errors.New("One or more errors occurred during host healthchecks")
	}

	return err
}

func ExecPush(mctx *common.MorphContext, hosts []nix.Host) (string, error) {
	sshCtx := ssh.CreateSSHContext(mctx.Options.SshOptions())

	resultPath, err := ExecBuild(mctx, hosts)
	if err != nil {
		return "", err
	}

	fmt.Fprintln(os.Stderr)
	return resultPath, pushPaths(sshCtx, hosts, resultPath)
}

func GetHosts(mctx *common.MorphContext, deploymentPath string) (deploymentMetadata nix.DeploymentMetadata, hosts []nix.Host, err error) {

	deploymentFile, err := os.Open(deploymentPath)
	if err != nil {
		return deploymentMetadata, hosts, err
	}

	deploymentAbsPath, err := filepath.Abs(deploymentFile.Name())
	if err != nil {
		return deploymentMetadata, hosts, err
	}

	deployment, err := nix.GetMachines(mctx.NixContext, deploymentAbsPath)
	if err != nil {
		return deploymentMetadata, hosts, err
	}

	matchingHosts, err := filter.MatchHosts(deployment.Hosts, mctx.Options.SelectGlob)
	if err != nil {
		return deploymentMetadata, hosts, err
	}

	var selectedTags []string
	if mctx.Options.SelectTags != "" {
		selectedTags = strings.Split(mctx.Options.SelectTags, ",")
	}

	matchingHosts2 := filter.FilterHostsTags(matchingHosts, selectedTags)

	ordering := deployment.Meta.Ordering
	if mctx.Options.OrderingTags != "" {
		ordering = nix.HostOrdering{Tags: strings.Split(mctx.Options.OrderingTags, ",")}
	}

	sortedHosts := filter.SortHosts(matchingHosts2, ordering)

	filteredHosts := filter.FilterHosts(sortedHosts, mctx.Options.SelectSkip, mctx.Options.SelectEvery, mctx.Options.SelectLimit)

	zLogHostsDict := zerolog.Dict()
	for _, host := range filteredHosts {
		zLogTagsArray := zerolog.Arr()
		for _, tag := range host.GetTags() {
			zLogTagsArray.Str(tag)
		}

		zLogHostsDict.Dict(
			host.Name,
			zerolog.Dict().
				Int("secrets", len(host.Secrets)).
				Int("health_checks", len(host.HealthChecks.Cmd)+len(host.HealthChecks.Http)).
				Array("tags", zLogTagsArray))

	}

	log.Info().
		Str("event", "deployment").
		Dict("hosts", zLogHostsDict).
		Msg("read deployment")

	return deployment.Meta, filteredHosts, nil
}

func activateConfiguration(mctx *common.MorphContext, filteredHosts []nix.Host, resultPath string) error {
	sshCtx := ssh.CreateSSHContext(mctx.Options.SshOptions())

	fmt.Fprintln(os.Stderr, "Executing '"+mctx.Options.DeploySwitchAction+"' on matched hosts:")
	fmt.Fprintln(os.Stderr)
	for _, host := range filteredHosts {

		fmt.Fprintln(os.Stderr, "** "+host.Name)

		configuration, err := nix.GetNixSystemPath(host, resultPath)
		if err != nil {
			return err
		}

		err = sshCtx.ActivateConfiguration(&host, configuration, mctx.Options.DeploySwitchAction)
		if err != nil {
			return err
		}

		fmt.Fprintln(os.Stderr)
	}

	return nil
}

func buildHosts(mctx *common.MorphContext, hosts []nix.Host) (resultPath string, err error) {
	if len(hosts) == 0 {
		err = errors.New("No hosts selected")
		return
	}

	deploymentPath, err := filepath.Abs(mctx.Options.Deployment)
	if err != nil {
		return
	}

	nixBuildTargets := ""
	if mctx.Options.NixBuildTargetFile != "" {
		if path, err := filepath.Abs(mctx.Options.NixBuildTargetFile); err == nil {
			nixBuildTargets = fmt.Sprintf("import \"%s\"", path)
		}
	} else if mctx.Options.NixBuildTarget != "" {
		nixBuildTargets = fmt.Sprintf("{ \"out\" = %s; }", mctx.Options.NixBuildTarget)
	}

	resultPath, err = nix.BuildMachines(mctx.NixContext, deploymentPath, hosts, mctx.Options.NixBuildArg, nixBuildTargets)

	if err != nil {
		return
	}

	log.Info().Msg("nix result path: " + resultPath)
	return
}

func pushPaths(sshContext *ssh.SSHContext, filteredHosts []nix.Host, resultPath string) error {
	for _, host := range filteredHosts {
		if host.BuildOnly {
			fmt.Fprintf(os.Stderr, "Push is disabled for build-only host: %s\n", host.Name)
			continue
		}

		paths, err := nix.GetPathsToPush(host, resultPath)
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Pushing paths to %v (%v@%v):\n", host.Name, host.TargetUser, host.TargetHost)
		for _, path := range paths {
			fmt.Fprintf(os.Stderr, "\t* %s\n", path)
		}
		err = nix.Push(sshContext, host, paths...)
		if err != nil {
			return err
		}
	}

	return nil
}
