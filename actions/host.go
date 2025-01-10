package actions

import (
	"errors"
	"github.com/DBCDK/morph/cache"
	"github.com/DBCDK/morph/common"
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

func (step IsOnline) Run(mctx *common.MorphContext, allHosts map[string]nix.Host, cache_ *cache.Cache) error {
	return errors.New("not implemented: " + step.Name())
}
func (step Reboot) Run(mctx *common.MorphContext, allHosts map[string]nix.Host, cache_ *cache.Cache) error {
	return errors.New("not implemented: " + step.Name())
}
