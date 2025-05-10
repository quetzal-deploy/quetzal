package steps

import (
	"context"

	"github.com/quetzal-deploy/quetzal/cache"
	"github.com/quetzal-deploy/quetzal/common"
	"github.com/quetzal-deploy/quetzal/nix"
)

type Action interface {
	Name() string
	MarshalJSONx(step Step) ([]byte, error)
	UnmarshalJSON(b []byte) error
	Run(ctx context.Context, opts *common.QuetzalOptions, hosts map[string]nix.Host, cache_ *cache.LockedMap[string]) error
}
