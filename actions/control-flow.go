package actions

import (
	"context"
	"errors"
	"github.com/DBCDK/morph/cache"
	"github.com/DBCDK/morph/common"
	"github.com/DBCDK/morph/nix"
)

type RepeatUntilSuccess struct {
	Period  int `json:"period"`
	Timeout int `json:"timeout"`
}

func (_ RepeatUntilSuccess) Name() string { return "repeat-until-success" }

func (step RepeatUntilSuccess) Run(ctx context.Context, mctx *common.MorphContext, allHosts map[string]nix.Host, cache_ *cache.LockedMap[string]) error {
	return errors.New("not implemented: " + step.Name())
	// FIXME: This should not be an action, but rather a behaviour setting for a step (`on-failure`)
	// FIXME: If action, maybe make Actions return (Step, err)
}
