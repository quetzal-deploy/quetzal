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

func ExecBuild(opts *common.MorphOptions, hosts []nix.Host) (string, error) {
	resultPath, err := buildHosts(opts, hosts)
	if err != nil {
		return "", err
	}
	return resultPath, nil
}

func ExecDeploy(opts *common.MorphOptions, hosts []nix.Host) (string, error) {
	sshCtx := ssh.CreateSSHContext(opts.SshOptions())

	doPush := false
	doUploadSecrets := false
	doActivate := false

	if !*opts.DryRun {
		switch opts.DeploySwitchAction {
		case "dry-activate":
			doPush = true
			doActivate = true
		case "test":
			fallthrough
		case "switch":
			fallthrough
		case "boot":
			doPush = true
			doUploadSecrets = opts.DeployUploadSecrets
			doActivate = true
		}
	}

	resultPath, err := buildHosts(opts, hosts)
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
			err = ExecUploadSecrets(opts, singleHostInList, &phase)
			if err != nil {
				return "", err
			}

			fmt.Fprintln(os.Stderr)
		}

		if !opts.SkipPreDeployChecks {
			err := healthchecks.PerformPreDeployChecks(sshCtx, &host, opts.Timeout)
			if err != nil {
				fmt.Fprintln(os.Stderr)
				fmt.Fprintln(os.Stderr, "Not deploying to additional hosts, since a host pre-deploy check failed.")
				utils.Exit(1)
			}
		}

		if doActivate {
			err = activateConfiguration(opts, singleHostInList, resultPath)
			if err != nil {
				return "", err
			}
		}

		if opts.DeployReboot {
			err = host.Reboot(sshCtx)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Reboot failed")
				return "", err
			}
		}

		if doUploadSecrets {
			phase := "post-activation"
			err = ExecUploadSecrets(opts, singleHostInList, &phase)
			if err != nil {
				return "", err
			}

			fmt.Fprintln(os.Stderr)
		}

		if !opts.SkipHealthChecks {
			err := healthchecks.PerformHealthChecks(sshCtx, &host, opts.Timeout)
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

func ExecEval(opts *common.MorphOptions) (string, error) {
	deploymentFile, err := os.Open(opts.Deployment)
	deploymentPath, err := filepath.Abs(deploymentFile.Name())
	if err != nil {
		return "", err
	}

	path, err := nix.EvalHosts(opts, deploymentPath, opts.AttrKey)

	return path, err
}

func ExecExecute(opts *common.MorphOptions, hosts []nix.Host) error {
	sshCtx := ssh.CreateSSHContext(opts.SshOptions())

	for _, host := range hosts {
		if host.BuildOnly {
			fmt.Fprintf(os.Stderr, "Exec is disabled for build-only host: %s\n", host.Name)
			continue
		}
		fmt.Fprintln(os.Stderr, "** "+host.Name)
		sshCtx.CmdInteractive(&host, opts.Timeout, opts.ExecuteCommand...)
		fmt.Fprintln(os.Stderr)
	}

	return nil
}

func ExecHealthCheck(opts *common.MorphOptions, hosts []nix.Host) error {
	sshCtx := ssh.CreateSSHContext(opts.SshOptions())

	var err error
	for _, host := range hosts {
		if host.BuildOnly {
			fmt.Fprintf(os.Stderr, "Healthchecks are disabled for build-only host: %s\n", host.Name)
			continue
		}
		err = healthchecks.PerformHealthChecks(sshCtx, &host, opts.Timeout)
	}

	if err != nil {
		err = errors.New("One or more errors occurred during host healthchecks")
	}

	return err
}

func ExecPush(opts *common.MorphOptions, hosts []nix.Host) (string, error) {
	sshCtx := ssh.CreateSSHContext(opts.SshOptions())

	resultPath, err := ExecBuild(opts, hosts)
	if err != nil {
		return "", err
	}

	fmt.Fprintln(os.Stderr)
	return resultPath, pushPaths(sshCtx, hosts, resultPath)
}

func GetHosts(opts *common.MorphOptions) (deploymentMetadata nix.DeploymentMetadata, hosts []nix.Host, err error) {

	deploymentFile, err := os.Open(opts.Deployment)
	if err != nil {
		return deploymentMetadata, hosts, err
	}

	deploymentAbsPath, err := filepath.Abs(deploymentFile.Name())
	if err != nil {
		return deploymentMetadata, hosts, err
	}

	deployment, err := nix.GetMachines(opts.NixOptions(), deploymentAbsPath)
	if err != nil {
		return deploymentMetadata, hosts, err
	}

	matchingHosts, err := filter.MatchHosts(deployment.Hosts, opts.SelectGlob)
	if err != nil {
		return deploymentMetadata, hosts, err
	}

	var selectedTags []string
	if opts.SelectTags != "" {
		selectedTags = strings.Split(opts.SelectTags, ",")
	}

	matchingHosts2 := filter.FilterHostsTags(matchingHosts, selectedTags)

	ordering := deployment.Meta.Ordering
	if opts.OrderingTags != "" {
		ordering = nix.HostOrdering{Tags: strings.Split(opts.OrderingTags, ",")}
	}

	sortedHosts := filter.SortHosts(matchingHosts2, ordering)

	filteredHosts := filter.FilterHosts(sortedHosts, opts.SelectSkip, opts.SelectEvery, opts.SelectLimit)

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

func activateConfiguration(opts *common.MorphOptions, filteredHosts []nix.Host, resultPath string) error {
	sshCtx := ssh.CreateSSHContext(opts.SshOptions())

	fmt.Fprintln(os.Stderr, "Executing '"+opts.DeploySwitchAction+"' on matched hosts:")
	fmt.Fprintln(os.Stderr)
	for _, host := range filteredHosts {

		fmt.Fprintln(os.Stderr, "** "+host.Name)

		configuration, err := nix.GetNixSystemPath(host, resultPath)
		if err != nil {
			return err
		}

		err = sshCtx.ActivateConfiguration(&host, configuration, opts.DeploySwitchAction)
		if err != nil {
			return err
		}

		fmt.Fprintln(os.Stderr)
	}

	return nil
}

func buildHosts(opts *common.MorphOptions, hosts []nix.Host) (resultPath string, err error) {
	if len(hosts) == 0 {
		err = errors.New("No hosts selected")
		return
	}

	deploymentPath, err := filepath.Abs(opts.Deployment)
	if err != nil {
		return
	}

	nixBuildTargets := ""
	if opts.NixBuildTargetFile != "" {
		if path, err := filepath.Abs(opts.NixBuildTargetFile); err == nil {
			nixBuildTargets = fmt.Sprintf("import \"%s\"", path)
		}
	} else if opts.NixBuildTarget != "" {
		nixBuildTargets = fmt.Sprintf("{ \"out\" = %s; }", opts.NixBuildTarget)
	}

	resultPath, err = nix.BuildMachines(opts, deploymentPath, hosts, opts.NixBuildArg, nixBuildTargets)

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
