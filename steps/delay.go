package steps

import (
	"context"
	"encoding/json"
	"time"

	"github.com/quetzal-deploy/quetzal/cache"
	"github.com/quetzal-deploy/quetzal/common"
	"github.com/quetzal-deploy/quetzal/nix"
)

type Delay struct {
	MilliSeconds int `json:"ms"`
}

func (delay Delay) Name() string {
	return "delay"
}

func (delay *Delay) MarshalJSONx(step Step) ([]byte, error) {
	type StepAlias Step

	return json.Marshal(struct {
		StepAlias
		Delay
	}{
		StepAlias: StepAlias(step),
		Delay:     *delay,
	})
}

func (delay *Delay) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, delay)
}

func (delay *Delay) Run(ctx context.Context, opts *common.QuetzalOptions, hosts map[string]nix.Host, cache_ *cache.LockedMap[string]) error {

	time.Sleep(time.Millisecond * time.Duration(delay.MilliSeconds))

	return nil
}
