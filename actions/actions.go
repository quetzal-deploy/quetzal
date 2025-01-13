package actions

import (
	"context"
	"github.com/DBCDK/morph/cache"
	"github.com/DBCDK/morph/common"
	"github.com/DBCDK/morph/nix"
)

type Action interface {
	Name() string
	Run(ctx context.Context, mctx *common.MorphContext, hosts map[string]nix.Host, cache_ *cache.LockedMap[string]) error // FIXME: look at morph-rs into what should be returned, and consider adding Step as parameter
}
