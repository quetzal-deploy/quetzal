package steps

import (
	"context"

	"github.com/quetzal-deploy/quetzal/internal/cache"
	"github.com/quetzal-deploy/quetzal/internal/common"
	"github.com/quetzal-deploy/quetzal/internal/nix"
)

type Action interface {
	Name() string
	MarshalJSONx(step Step) ([]byte, error)
	UnmarshalJSON(b []byte) error
	Run(ctx context.Context, opts *common.QuetzalOptions, hosts map[string]nix.Host, cache_ *cache.LockedMap[string]) error
}
