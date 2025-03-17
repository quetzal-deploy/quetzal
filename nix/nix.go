package nix

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/DBCDK/morph/common"
	"github.com/DBCDK/morph/logging"
	"github.com/rs/zerolog/log"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/DBCDK/morph/healthchecks"
	"github.com/DBCDK/morph/secrets"
	"github.com/DBCDK/morph/ssh"
	"github.com/DBCDK/morph/utils"
)

type Host struct {
	PreDeployChecks         healthchecks.HealthChecks
	HealthChecks            healthchecks.HealthChecks
	Name                    string `json:"name"`
	NixosRelease            string
	TargetHost              string
	TargetPort              int
	TargetUser              string
	Secrets                 map[string]secrets.Secret
	BuildOnly               bool
	SubstituteOnDestination bool
	NixConfig               map[string]string
	Tags                    []string
	Labels                  map[string]string `json:"labels"`
}

type HostOrdering struct {
	Tags []string
}

type Constraint struct {
	Selector       LabelSelector `json:"selector"`
	MaxUnavailable int           `json:"maxUnavailable"`
}

func NewConstraint(selector LabelSelector, maxUnavailable int) Constraint {
	return Constraint{
		Selector:       selector,
		MaxUnavailable: maxUnavailable,
	}
}

// Set default values for a Constraint when being unmarshalled
func (c *Constraint) UnmarshalJSON(data []byte) error {
	type ConstraintAlias Constraint
	return json.Unmarshal(data, (*ConstraintAlias)(c))
}

// FIXME: Make LabelSelector a map[string]string where all labels have to match
type LabelSelector struct {
	Label string
	Value string
}

func (ls LabelSelector) Match(label string, value string) bool {
	return ls.Label == label && (ls.Value == "*" || value == "*" || ls.Value == value)
}

type DeploymentMetadata struct {
	Description string
	Ordering    HostOrdering
	Constraints []Constraint
	Color       string `json:"color"`
}

type Deployment struct {
	Hosts []Host             `json:"hosts"`
	Meta  DeploymentMetadata `json:"meta"`
}

type NixBuildInvocationArgs struct {
	ArgsFile        string
	Attr            string
	DeploymentPath  string
	Names           []string
	NixArgs         []string
	NixBuildTargets string
	NixConfig       map[string]string
	NixOptions      common.NixOptions
	ResultLinkPath  string
}

func (nArgs *NixBuildInvocationArgs) ToNixBuildArgs() []string {
	args := []string{
		nArgs.NixOptions.EvalMachines,
		"--arg", "networkExpr", nArgs.DeploymentPath,
		"--argstr", "argsFile", nArgs.ArgsFile,
		"--out-link", nArgs.ResultLinkPath,
		"--attr", nArgs.Attr,
	}

	args = append(args, mkOptions(nArgs.NixConfig)...)

	if len(nArgs.NixArgs) > 0 {
		args = append(args, nArgs.NixArgs...)
	}

	if nArgs.NixOptions.ShowTrace {
		args = append(args, "--show-trace")
	}

	if nArgs.NixBuildTargets != "" {
		args = append(args,
			"--arg", "buildTargets", nArgs.NixBuildTargets)
	}

	return args
}

func (nArgs *NixEvalInvocationArgs) ToNixInstantiateArgs() []string {
	args := []string{
		"--eval", nArgs.NixOptions.EvalMachines,
		"--arg", "networkExpr", nArgs.DeploymentPath,
		"--argstr", "argsFile", nArgs.ArgsFile,
		"--attr", nArgs.Attr,
	}

	if nArgs.NixOptions.ShowTrace {
		args = append(args, "--show-trace")
	}

	if nArgs.AsJSON {
		args = append(args, "--json")
	}

	if nArgs.Strict {
		args = append(args, "--strict")
	}

	if nArgs.ReadWriteMode {
		args = append(args, "--read-write-mode")
	}

	return args
}

type NixEvalInvocationArgs struct {
	AsJSON         bool
	ArgsFile       string
	Attr           string
	DeploymentPath string
	NixOptions     common.NixOptions
	Strict         bool
	ReadWriteMode  bool
}

func (host *Host) GetName() string {
	return host.Name
}

func (host *Host) GetTargetHost() string {
	return host.TargetHost
}

func (host *Host) GetTargetPort() int {
	return host.TargetPort
}

func (host *Host) GetTargetUser() string {
	return host.TargetUser
}

func (host *Host) GetHealthChecks() healthchecks.HealthChecks {
	return host.HealthChecks
}

func (host *Host) GetPreDeployChecks() healthchecks.HealthChecks {
	return host.PreDeployChecks
}

func (host *Host) GetTags() []string {
	return host.Tags
}

func (host *Host) Reboot(sshContext *ssh.SSHContext) error {

	var (
		oldBootID string
		newBootID string
	)

	oldBootID, err := sshContext.GetBootID(host)
	// If the host doesn't support getting boot ID's for some reason, warn about it, and skip the comparison
	skipBootIDComparison := err != nil
	if skipBootIDComparison {
		fmt.Fprintf(os.Stderr, "Error getting boot ID (this is used to determine when the reboot is complete): %v\n", err)
		fmt.Fprintf(os.Stderr, "This makes it impossible to detect when the host has rebooted, so health checks might pass before the host has rebooted.\n")
	}

	if cmd, err := sshContext.Cmd(host, "sudo", "reboot"); cmd != nil {
		log.Info().Msg("Asking host to reboot ... ")

		if err = cmd.Run(); err != nil {
			// Here we assume that exit code 255 means: "SSH connection got disconnected",
			// which is OK for a reboot - sshd may close active connections before we disconnect after all
			if exitErr, ok := err.(*exec.ExitError); ok {
				if status, ok := exitErr.Sys().(syscall.WaitStatus); ok && status.ExitStatus() == 255 {
					fmt.Fprintln(os.Stderr, "Remote host disconnected.")
					err = nil
				}
			}
		}

		if err != nil {
			log.Info().Msg("Asking host to reboot ...: Failed ")
			return err
		}
	}

	log.Info().Msg("Asking host to reboot ...: OK ")

	if !skipBootIDComparison {
		log.Info().Msg("Waiting for host to come online ")

		// Wait for the host to get a new boot ID. These ID's should be unique for each boot,
		// meaning a reboot will have been completed when the boot ID has changed.
		for {
			log.Info().Msg("Waiting for host to come online ")

			// Ignore errors; there'll be plenty of them since we'll be attempting to connect to an offline host,
			// and we know from previously that the host should support boot ID's
			newBootID, _ = sshContext.GetBootID(host)

			if newBootID != "" && oldBootID != newBootID {
				log.Info().Msg("Waiting for host to come online: OK")
				break
			}

			time.Sleep(2 * time.Second)
		}
	}

	return nil
}

func GetBuildShell(nixOptions *common.NixOptions, deploymentPath string) (buildShell *string, err error) {

	nixEvalInvocationArgs := NixEvalInvocationArgs{
		AsJSON:         true,
		Attr:           "info.buildShell",
		DeploymentPath: deploymentPath,
		NixOptions:     *nixOptions,
		Strict:         true,
	}

	jsonArgs, err := json.Marshal(nixEvalInvocationArgs)
	if err != nil {
		return buildShell, err
	}

	cmd := exec.Command(nixOptions.EvalCmd, nixEvalInvocationArgs.ToNixInstantiateArgs()...)

	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = logging.CmdWriter{Host: ""}

	utils.AddFinalizer(func() {
		if (cmd.ProcessState == nil || !cmd.ProcessState.Exited()) && cmd.Process != nil {
			_ = cmd.Process.Signal(syscall.SIGTERM)
		}
	})
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf("MORPH_ARGS=%s", jsonArgs))
	err = cmd.Run()
	if err != nil {
		errorMessage := fmt.Sprintf(
			"Error while running `%s ..`: %s", nixOptions.EvalCmd, err.Error(),
		)
		return buildShell, errors.New(errorMessage)
	}

	err = json.Unmarshal(stdout.Bytes(), &buildShell)
	if err != nil {
		return nil, err
	}

	return buildShell, nil
}

func EvalHosts(opts *common.MorphOptions, deploymentPath string, attr string) (string, error) {
	nixOptions := opts.NixOptions()

	attribute := "nodes." + attr

	nixEvalInvocationArgs := NixEvalInvocationArgs{
		AsJSON:         false,
		Attr:           attribute,
		DeploymentPath: deploymentPath,
		NixOptions:     *nixOptions,
		Strict:         true,
	}

	jsonArgs, err := json.Marshal(nixEvalInvocationArgs)
	if err != nil {
		return "", err
	}

	cmd := exec.Command(nixOptions.EvalCmd, nixEvalInvocationArgs.ToNixInstantiateArgs()...)

	utils.AddFinalizer(func() {
		if (cmd.ProcessState == nil || !cmd.ProcessState.Exited()) && cmd.Process != nil {
			_ = cmd.Process.Signal(syscall.SIGTERM)
		}
	})

	logging.LogCmd("localhost", cmd)

	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf("MORPH_ARGS=%s", jsonArgs))
	err = cmd.Run()
	return deploymentPath, err
}

func GetMachines(nixOptions *common.NixOptions, deploymentPath string) (deployment Deployment, err error) {

	nixEvalInvocationArgs := NixEvalInvocationArgs{
		AsJSON:         true,
		Attr:           "info.deployment",
		DeploymentPath: deploymentPath,
		NixOptions:     *nixOptions,
		Strict:         true,
	}

	jsonArgs, err := json.Marshal(nixEvalInvocationArgs)
	if err != nil {
		return deployment, err
	}

	cmd := exec.Command(nixOptions.EvalCmd, nixEvalInvocationArgs.ToNixInstantiateArgs()...)

	var stdout bytes.Buffer
	cmd.Stdout = &stdout // FIXME: zlog this while still capturing stdout
	cmd.Stderr = logging.CmdWriter{Host: ""}

	utils.AddFinalizer(func() {
		if (cmd.ProcessState == nil || !cmd.ProcessState.Exited()) && cmd.Process != nil {
			_ = cmd.Process.Signal(syscall.SIGTERM)
		}
	})
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf("MORPH_ARGS=%s", jsonArgs))
	err = cmd.Run()
	if err != nil {
		errorMessage := fmt.Sprintf(
			"Error while running `%s ..`: %s", nixOptions.EvalCmd, err.Error(),
		)
		return deployment, errors.New(errorMessage)
	}

	err = json.Unmarshal(stdout.Bytes(), &deployment)
	if err != nil {
		return deployment, err
	}

	return deployment, nil
}

func BuildMachines(opts *common.MorphOptions, deploymentPath string, hosts []Host, nixArgs []string, nixBuildTargets string) (resultPath string, err error) {
	nixOptions := opts.NixOptions()

	tmpdir, err := ioutil.TempDir("", "morph-")
	if err != nil {
		return "", err
	}
	utils.AddFinalizer(func() {
		os.RemoveAll(tmpdir)
	})

	hostNames := []string{}
	for _, host := range hosts {
		hostNames = append(hostNames, host.Name)
	}

	resultLinkPath := filepath.Join(path.Dir(deploymentPath), ".gcroots", path.Base(deploymentPath))
	if nixOptions.KeepGCRoot {
		if err = os.MkdirAll(path.Dir(resultLinkPath), 0755); err != nil {
			nixOptions.KeepGCRoot = false
			fmt.Fprintf(os.Stderr, "Unable to create GC root, skipping: %s", err)
		}
	}
	if !nixOptions.KeepGCRoot {
		// create tmp dir for result link
		resultLinkPath = filepath.Join(tmpdir, "result")
	}

	buildShell, err := GetBuildShell(nixOptions, deploymentPath)

	if err != nil {
		errorMessage := fmt.Sprintf(
			"Error getting buildShell.",
		)
		return resultPath, errors.New(errorMessage)
	}

	argsFile := tmpdir + "/morph-args.json"
	NixBuildInvocationArgs := NixBuildInvocationArgs{
		ArgsFile:        argsFile,
		Attr:            "machines",
		DeploymentPath:  deploymentPath,
		Names:           hostNames,
		NixArgs:         nixArgs,
		NixBuildTargets: nixBuildTargets,
		NixConfig:       hosts[0].NixConfig,
		NixOptions:      *nixOptions,
		ResultLinkPath:  resultLinkPath,
	}

	jsonArgs, err := json.Marshal(NixBuildInvocationArgs)
	if err != nil {
		return "", err
	}

	err = ioutil.WriteFile(argsFile, jsonArgs, 0644)
	if err != nil {
		return "", err
	}

	var cmd *exec.Cmd
	if nixOptions.AllowBuildShell && buildShell != nil {
		shellArgs := strings.Join(append([]string{nixOptions.BuildCmd}, NixBuildInvocationArgs.ToNixBuildArgs()...), " ")
		cmd = exec.Command(nixOptions.ShellCmd, *buildShell, "--pure", "--run", shellArgs)
	} else {
		cmd = exec.Command(nixOptions.BuildCmd, NixBuildInvocationArgs.ToNixBuildArgs()...)
	}

	utils.AddFinalizer(func() {
		if (cmd.ProcessState == nil || !cmd.ProcessState.Exited()) && cmd.Process != nil {
			_ = cmd.Process.Signal(syscall.SIGTERM)
		}
	})

	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf("MORPH_ARGS=%s", jsonArgs))
	cmd.Env = append(cmd.Env, fmt.Sprintf("MORPH_ARGS_FILE=%s", argsFile))

	logging.LogCmd("localhost", cmd)
	err = cmd.Run()

	if err != nil {
		errorMessage := fmt.Sprintf(
			"Error while running `%s ...`: See above.", cmd.String(),
		)
		return resultPath, errors.New(errorMessage)
	}

	resultPath, err = os.Readlink(resultLinkPath)
	if err != nil {
		return "", err
	}

	return
}

func mkOptionsFromHost(host Host) []string {
	return mkOptions(host.NixConfig)
}

func mkOptions(nixConfig map[string]string) []string {
	var options = make([]string, 0)
	for k, v := range nixConfig {
		options = append(options, "--option")
		options = append(options, k)
		options = append(options, v)
	}
	return options
}

func GetNixSystemPath(host Host, resultPath string) (string, error) {
	return os.Readlink(filepath.Join(resultPath, host.Name))
}

func GetNixSystemDerivation(host Host, resultPath string) (string, error) {
	return os.Readlink(filepath.Join(resultPath, host.Name+".drv"))
}

func GetPathsToPush(host Host, resultPath string) (paths []string, err error) {
	path1, err := GetNixSystemPath(host, resultPath)
	if err != nil {
		return paths, err
	}

	paths = append(paths, path1)

	return paths, nil
}

func Push(sshCtx *ssh.SSHContext, host Host, paths ...string) (err error) {
	utils.ValidateEnvironment("ssh")

	var userArg = ""
	var keyArg = ""
	var sshOpts = []string{}
	var env = os.Environ()
	if host.TargetUser != "" {
		userArg = host.TargetUser + "@"
	} else if sshCtx.SshOptions.DefaultUsername != "" {
		userArg = sshCtx.SshOptions.DefaultUsername + "@"
	}
	if sshCtx.SshOptions.IdentityFile != "" {
		keyArg = "?ssh-key=" + sshCtx.SshOptions.IdentityFile
	}
	if sshCtx.SshOptions.SkipHostKeyCheck {
		sshOpts = append(sshOpts, fmt.Sprintf("%s", "-o StrictHostkeyChecking=No -o UserKnownHostsFile=/dev/null"))
	}
	if host.TargetPort != 0 {
		sshOpts = append(sshOpts, fmt.Sprintf("-p %d", host.TargetPort))
	}
	if sshCtx.SshOptions.ConfigFile != "" {
		sshOpts = append(sshOpts, fmt.Sprintf("-F %s", sshCtx.SshOptions.ConfigFile))
	}
	if len(sshOpts) > 0 {
		env = append(env, fmt.Sprintf("NIX_SSHOPTS=%s", strings.Join(sshOpts, " ")))
	}

	options := mkOptionsFromHost(host)
	for _, path := range paths {
		args := []string{
			"--to", userArg + host.TargetHost + keyArg,
			path,
		}
		args = append(args, options...)
		if host.SubstituteOnDestination {
			args = append(args, "--use-substitutes")
		}

		cmd := exec.Command(
			"nix-copy-closure", args...,
		)
		logging.LogCmd(host.TargetHost, cmd)

		cmd.Env = env

		err = cmd.Run()

		if err != nil {
			return err
		}
	}

	return nil
}
