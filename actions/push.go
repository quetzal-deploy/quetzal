package actions

import (
	"context"
	"fmt"
	"github.com/DBCDK/morph/cache"
	"github.com/DBCDK/morph/common"
	"github.com/DBCDK/morph/nix"
	"github.com/rs/zerolog/log"
)

type Push struct {
	Host string `json:"host"`
}

func (_ Push) Name() string {
	return "push"
}

func (step Push) Run(ctx context.Context, mctx *common.MorphContext, hosts map[string]nix.Host, cache_ *cache.LockedMap[string]) error {
	cacheKey := "closure:" + step.Host
	log.Debug().Msg("cache key: " + cacheKey)
	closure, err := cache_.Get(cacheKey)
	if err != nil {
		return err
	}

	log.Info().Msg(fmt.Sprintf("Pushing %s to %s\n", closure, hosts[step.Host].TargetHost))

	err = nix.Push(mctx.SSHContext, hosts[step.Host], closure)

	return err
}
