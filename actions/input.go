package actions

import (
	"context"
	"errors"
	"github.com/DBCDK/morph/cache"
	"github.com/DBCDK/morph/common"
	"github.com/DBCDK/morph/nix"
)

type GetSudoPasswd struct{}

func (_ GetSudoPasswd) Name() string { return "get-sudo-password" }

func (step GetSudoPasswd) Run(ctx context.Context, mctx *common.MorphContext, allHosts map[string]nix.Host, cache_ *cache.LockedMap[string]) error {
	return errors.New("not implemented: " + step.Name())
}
