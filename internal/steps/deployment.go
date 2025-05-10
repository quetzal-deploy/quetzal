package steps

import (
	"context"
	"errors"

	"github.com/quetzal-deploy/quetzal/internal/cache"
	"github.com/quetzal-deploy/quetzal/internal/common"
	"github.com/quetzal-deploy/quetzal/internal/nix"
)

type EvalDeployment struct {
	Deployment string `json:"deployment"`
}

func (_ EvalDeployment) Name() string { return "eval-deployment" }

func (step EvalDeployment) Run(ctx context.Context, opts *common.QuetzalOptions, allHosts map[string]nix.Host, cache_ *cache.LockedMap[string]) error {
	return errors.New("not implemented: " + step.Name())
}
