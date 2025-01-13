package actions

import (
	"context"
	"errors"
	"github.com/DBCDK/morph/cache"
	"github.com/DBCDK/morph/common"
	"github.com/DBCDK/morph/nix"
)

type LocalCommand struct {
	Command []string `json:"command"`
	Timeout int      `json:"timeout"`
}

type RemoteCommand struct {
	Command []string `json:"command"`
	Timeout int      `json:"timeout"`
}

func (_ LocalCommand) Name() string  { return "local-command" }
func (_ RemoteCommand) Name() string { return "remote-command" }

func (step LocalCommand) Run(ctx context.Context, mctx *common.MorphContext, allHosts map[string]nix.Host, cache_ *cache.LockedMap[string]) error {
	return errors.New("not implemented: " + step.Name())
}

func (step RemoteCommand) Run(ctx context.Context, mctx *common.MorphContext, allHosts map[string]nix.Host, cache_ *cache.LockedMap[string]) error {
	return errors.New("not implemented: " + step.Name())
}
