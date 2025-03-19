package common

import (
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"

	"github.com/DBCDK/morph/utils"
)

type MorphOptions struct {
	Version   string
	AssetRoot string

	DryRun          *bool
	JsonOut         *bool
	ConstraintsFlag *[]string
	KeepGCRoot      *bool
	AllowBuildShell *bool
	PlanOnly        *bool
	DotFile         *string

	AsJson              bool
	AskForSudoPasswd    bool
	AttrKey             string
	Deployment          string
	DeploymentsDir      string
	DeployReboot        bool
	DeploySwitchAction  string
	DeployUploadSecrets bool
	ExecuteCommand      []string
	NixBuildArg         []string
	NixBuildTarget      string
	NixBuildTargetFile  string
	OrderingTags        string
	PassCmd             string
	PlanAction          string
	PlanFile            string
	SelectEvery         int
	SelectGlob          string
	SelectLimit         int
	SelectSkip          int
	SelectTags          string
	ShowTrace           bool
	SkipHealthChecks    bool
	SkipPreDeployChecks bool
	Timeout             int
}

type NixOptions struct {
	EvalCmd         string
	BuildCmd        string
	ShellCmd        string
	EvalMachines    string
	ShowTrace       bool
	KeepGCRoot      bool
	AllowBuildShell bool
}

type SshOptions struct {
	SudoPassword           string
	AskForSudoPassword     bool
	GetSudoPasswordCommand string
	DefaultUsername        string
	IdentityFile           string
	ConfigFile             string
	SkipHostKeyCheck       bool
}

func (o *MorphOptions) NixOptions() *NixOptions {
	evalCmd := os.Getenv("MORPH_NIX_EVAL_CMD")
	buildCmd := os.Getenv("MORPH_NIX_BUILD_CMD")
	shellCmd := os.Getenv("MORPH_NIX_SHELL_CMD")
	evalMachines := os.Getenv("MORPH_NIX_EVAL_MACHINES")

	if evalCmd == "" {
		evalCmd = "nix-instantiate"
	}
	if buildCmd == "" {
		buildCmd = "nix-build"
	}
	if shellCmd == "" {
		shellCmd = "nix-shell"
	}
	if evalMachines == "" {
		evalMachines = filepath.Join(o.AssetRoot, "eval-machines.nix")
	}

	return &NixOptions{
		EvalCmd:         evalCmd,
		BuildCmd:        buildCmd,
		ShellCmd:        shellCmd,
		EvalMachines:    evalMachines,
		ShowTrace:       o.ShowTrace,
		KeepGCRoot:      *o.KeepGCRoot,
		AllowBuildShell: *o.AllowBuildShell,
	}
}

func (o *MorphOptions) SshOptions() *SshOptions {
	return &SshOptions{
		AskForSudoPassword:     o.AskForSudoPasswd,
		GetSudoPasswordCommand: o.PassCmd,
		IdentityFile:           os.Getenv("SSH_IDENTITY_FILE"),
		DefaultUsername:        os.Getenv("SSH_USER"),
		SkipHostKeyCheck:       os.Getenv("SSH_SKIP_HOST_KEY_CHECK") != "",
		ConfigFile:             os.Getenv("SSH_CONFIG_FILE"),
	}
}

func HandleError(err error) {
	//Stupid handling of catch-all errors for now
	if err != nil {
		log.Fatal().Err(err).Msg("Fatal error, shutting down")
		utils.Exit(1)
	}
}
