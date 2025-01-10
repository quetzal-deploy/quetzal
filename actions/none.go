package actions

import (
	"github.com/DBCDK/morph/cache"
	"github.com/DBCDK/morph/common"
	"github.com/DBCDK/morph/nix"
)

type None struct{}
type Gate None
type Wrapper None

func (_ None) Name() string    { return "none" }
func (_ Gate) Name() string    { return "gate" }
func (_ Wrapper) Name() string { return "gate" }

func (step None) Run(mctx *common.MorphContext, allHosts map[string]nix.Host, cache_ *cache.Cache) error {
	return nil
}

func (step Gate) Run(mctx *common.MorphContext, allHosts map[string]nix.Host, cache_ *cache.Cache) error {
	return nil
}

func (step Wrapper) Run(mctx *common.MorphContext, allHosts map[string]nix.Host, cache_ *cache.Cache) error {
	return nil
}
