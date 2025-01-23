package actions

import (
	"context"
	"errors"
	"fmt"
	"github.com/DBCDK/morph/cache"
	"github.com/DBCDK/morph/common"
	"github.com/DBCDK/morph/logging"
	"github.com/DBCDK/morph/nix"
)

type IsOnline struct {
	Host string `json:"host"`
}

type Reboot struct {
	Host string `json:"host"`
}

func (_ IsOnline) Name() string { return "is-online" }
func (_ Reboot) Name() string   { return "reboot" }

func (step IsOnline) Run(ctx context.Context, mctx *common.MorphContext, allHosts map[string]nix.Host, cache_ *cache.LockedMap[string]) error {
	host, ok := allHosts[step.Host]
	if !ok {
		return errors.New(fmt.Sprintf("host '%s' not in deployment", step.Host))
	}

	cmd, err := mctx.SSHContext.CmdContext(ctx, &host, "/bin/sh", "-c", "true")
	if err != nil {
		return err
	}

	logging.LogCmd(step.Host, cmd)

	err = cmd.Run()

	return err
}
func (step Reboot) Run(ctx context.Context, mctx *common.MorphContext, allHosts map[string]nix.Host, cache_ *cache.LockedMap[string]) error {
	host, exists := allHosts[step.Host]
	if !exists {
		return errors.New("unknown host: " + step.Host)
	}

	err := host.Reboot(mctx.SSHContext)

	return err
}
