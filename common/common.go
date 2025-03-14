package common

import (
	"github.com/DBCDK/morph/nix"
	"github.com/DBCDK/morph/ssh"
	"github.com/DBCDK/morph/utils"
	"github.com/rs/zerolog/log"
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
	HostsMap            map[string]nix.Host
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

type MorphContext struct {
	Config     *MorphOptions
	SSHContext *ssh.SSHContext
	NixContext *nix.NixContext
}

func HandleError(err error) {
	//Stupid handling of catch-all errors for now
	if err != nil {
		log.Fatal().Err(err).Msg("Fatal error, shutting down")
		utils.Exit(1)
	}
}
