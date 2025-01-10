package actions

import (
	"github.com/DBCDK/morph/cache"
	"github.com/DBCDK/morph/common"
	"github.com/DBCDK/morph/nix"
)

type Action interface {
	Name() string
	Run(mctx *common.MorphContext, hosts map[string]nix.Host, cache_ *cache.Cache) error // FIXME: look at morph-rs into what should be returned, and consider adding Step as parameter
}
