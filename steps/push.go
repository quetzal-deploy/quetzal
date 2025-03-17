package steps

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/DBCDK/morph/cache"
	"github.com/DBCDK/morph/common"
	"github.com/DBCDK/morph/nix"
	"github.com/DBCDK/morph/ssh"
)

type Push struct {
	Host string `json:"host"`
}

func (push Push) Name() string {
	return "push"
}

func (push *Push) MarshalJSONx(step Step) ([]byte, error) {
	type StepAlias Step

	return json.Marshal(struct {
		StepAlias
		Push
	}{
		StepAlias: StepAlias(step),
		Push:      *push,
	})
}

func (push *Push) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, push)
}

func (push *Push) Run(ctx context.Context, opts *common.MorphOptions, hosts map[string]nix.Host, cache_ *cache.LockedMap[string]) error {
	sshContext := ssh.CreateSSHContext(opts)

	cacheKey := "closure:" + push.Host
	log.Debug().Msg("cache key: " + cacheKey)
	closure, err := cache_.Get(cacheKey)
	if err != nil {
		return err
	}

	log.Info().Msg(fmt.Sprintf("Pushing %s to %s\n", closure, hosts[push.Host].TargetHost))

	err = nix.Push(sshContext, hosts[push.Host], closure)

	return err
}
