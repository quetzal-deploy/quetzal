package actions

import (
	"errors"
	"fmt"
	"github.com/DBCDK/morph/cache"
	"github.com/DBCDK/morph/common"
	"github.com/DBCDK/morph/cruft"
	"github.com/DBCDK/morph/nix"
	"path"
	"path/filepath"
)

type Build struct {
	Hosts []string `json:"hosts"`
}

func (_ Build) Name() string {
	return "build"
}

func filterHosts(needles []string, allHosts map[string]nix.Host) ([]nix.Host, error) {
	result := make([]nix.Host, 0)

	for _, hostByName := range needles {
		host, ok := allHosts[hostByName]
		if ok {
			result = append(result, host)
		} else {
			return nil, errors.New(fmt.Sprintf("host %s not in deployment", hostByName))
		}
	}

	return result, nil
}

func (step Build) Run(mctx *common.MorphContext, allHosts map[string]nix.Host, cache_ *cache.Cache) error {
	hosts, err := filterHosts(step.Hosts, allHosts)
	if err != nil {
		return err
	}

	resultPath, err := cruft.ExecBuild(mctx, hosts)
	if err != nil {
		return err
	}

	fmt.Println(resultPath)

	for _, host := range hosts {
		hostPathSymlink := path.Join(resultPath, host.Name)
		hostPath, err := filepath.EvalSymlinks(hostPathSymlink)
		if err != nil {
			return err
		}

		fmt.Println(hostPathSymlink)
		fmt.Println(hostPath)

		cache_.Update(cache.StepData{Key: "closure:" + host.Name, Value: hostPath})
	}

	return err
}
