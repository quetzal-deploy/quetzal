package actions

import (
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

func (step RepeatUntilSuccess) Run(mctx *common.MorphContext, allHosts map[string]nix.Host, cache_ *cache.Cache) error {
	return errors.New("not implemented: " + step.Name())
}
