package common

import (
	"fmt"
	"os"

	"github.com/DBCDK/morph/nix"
	"github.com/DBCDK/morph/ssh"
	"github.com/DBCDK/morph/utils"
)

type MorphContext struct {
	SSHContext *ssh.SSHContext
	NixContext *nix.NixContext

	// Globals we need to get rid off
	AssetRoot           string
	AttrKey             string
	Deployment          string
	DeploySwitchAction  string
	DeployReboot        bool
	DeployUploadSecrets bool
	DryRun              bool
	ExecuteCommand      []string
	NixBuildArg         []string
	NixBuildTarget      string
	NixBuildTargetFile  string
	OrderingTags        string
	SelectEvery         int
	SelectGlob          string
	SelectLimit         int
	SelectSkip          int
	SelectTags          string
	SkipHealthChecks    bool
	SkipPreDeployChecks bool
	Timeout             int
}

func HandleError(err error) {
	//Stupid handling of catch-all errors for now
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		utils.Exit(1)
	}
}

type StepUpdateEvent struct {
	StepId string
	State  string
}

// FIXME: Merge this with LogEvent
type StepLogEvent struct {
	StepId string
	Data   string
}
