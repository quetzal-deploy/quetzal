package actions

import (
	"fmt"
	"github.com/DBCDK/morph/cache"
	"github.com/DBCDK/morph/common"
	"github.com/DBCDK/morph/nix"
)

type Push struct {
	Host string `json:"host"`
}

func (_ Push) Name() string {
	return "push"
}

func (step Push) Run(mctx *common.MorphContext, hosts map[string]nix.Host, cache_ *cache.Cache) error {
	cacheKey := "closure:" + step.Host
	fmt.Println("cache key: " + cacheKey)
	closure, err := cache_.Get(cacheKey)
	if err != nil {
		return err
	}

	fmt.Printf("Pushing %s to %s\n", closure, hosts[step.Host].TargetHost)

	err = nix.Push(mctx.SSHContext, hosts[step.Host], closure)

	return err
}
