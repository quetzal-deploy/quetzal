package actions

import (
	"errors"
	"github.com/DBCDK/morph/cache"
	"github.com/DBCDK/morph/common"
	"github.com/DBCDK/morph/nix"
)

type DeployBoot struct{ ActionWithOneHost }
type DeployDryActivate struct{ ActionWithOneHost }
type DeploySwitch struct{ ActionWithOneHost }
type DeployTest struct{ ActionWithOneHost }

func (_ DeployBoot) Name() string        { return "deploy-boot" }
func (_ DeployDryActivate) Name() string { return "deploy-dry-activate" }
func (_ DeploySwitch) Name() string      { return "deploy-switch" }
func (_ DeployTest) Name() string        { return "deploy-test" }

func (step DeployBoot) Run(mctx *common.MorphContext, allHosts map[string]nix.Host, cache_ *cache.Cache) error {
	return errors.New("not implemented: " + step.Name())
}

func (step DeployDryActivate) Run(mctx *common.MorphContext, allHosts map[string]nix.Host, cache_ *cache.Cache) error {
	return errors.New("not implemented: " + step.Name())
}

func (step DeploySwitch) Run(mctx *common.MorphContext, allHosts map[string]nix.Host, cache_ *cache.Cache) error {
	return errors.New("not implemented: " + step.Name())
}

func (step DeployTest) Run(mctx *common.MorphContext, allHosts map[string]nix.Host, cache_ *cache.Cache) error {
	return errors.New("not implemented: " + step.Name())
}
