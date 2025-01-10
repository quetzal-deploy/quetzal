package actions

import (
	"errors"
	"fmt"
	"github.com/DBCDK/morph/cache"
	"github.com/DBCDK/morph/common"
	"github.com/DBCDK/morph/nix"
	"os"
)

type DeployBoot struct {
	Host string `json:"host"`
}

type DeployDryActivate struct {
	Host string `json:"host"`
}

type DeploySwitch struct {
	Host string `json:"host"`
}

type DeployTest struct {
	Host string `json:"host"`
}

func (_ DeployBoot) Name() string        { return "deploy-boot" }
func (_ DeployDryActivate) Name() string { return "deploy-dry-activate" }
func (_ DeploySwitch) Name() string      { return "deploy-switch" }
func (_ DeployTest) Name() string        { return "deploy-test" }

func (step DeployBoot) Run(mctx *common.MorphContext, allHosts map[string]nix.Host, cache_ *cache.Cache) error {
	host, ok := allHosts[step.Host]
	if !ok {
		return errors.New(fmt.Sprintf("host '%s' not in deployment", step.Host))
	}

	return deploy(mctx, cache_, host, "test")
}

func (step DeployDryActivate) Run(mctx *common.MorphContext, allHosts map[string]nix.Host, cache_ *cache.Cache) error {
	host, ok := allHosts[step.Host]
	if !ok {
		return errors.New(fmt.Sprintf("host '%s' not in deployment", step.Host))
	}

	return deploy(mctx, cache_, host, "test")
}

func (step DeploySwitch) Run(mctx *common.MorphContext, allHosts map[string]nix.Host, cache_ *cache.Cache) error {
	host, ok := allHosts[step.Host]
	if !ok {
		return errors.New(fmt.Sprintf("host '%s' not in deployment", step.Host))
	}

	return deploy(mctx, cache_, host, "test")
}

func (step DeployTest) Run(mctx *common.MorphContext, allHosts map[string]nix.Host, cache_ *cache.Cache) error {
	host, ok := allHosts[step.Host]
	if !ok {
		return errors.New(fmt.Sprintf("host '%s' not in deployment", step.Host))
	}

	return deploy(mctx, cache_, host, "test")
}

func deploy(mctx *common.MorphContext, cache_ *cache.Cache, host nix.Host, deployAction string) error {
	fmt.Fprintf(os.Stderr, "Executing %s on %s", deployAction, host.Name)

	closure, err := cache_.Get("closure:" + host.Name)
	if err != nil {
		return err
	}

	err = mctx.SSHContext.ActivateConfiguration(&host, closure, deployAction)
	if err != nil {
		return err
	}

	return nil
}
