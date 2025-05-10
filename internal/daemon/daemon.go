package daemon

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/quetzal-deploy/quetzal/common"
	"github.com/quetzal-deploy/quetzal/cruft"
	"github.com/quetzal-deploy/quetzal/nix"
)

type Daemon struct {
	Deployments map[string]Deployment
	//EventManager *events.Manager

	quetzalOptions  *common.QuetzalOptions
	deploymentsPath string
}

func NewDaemon(opts *common.QuetzalOptions) Daemon {
	return Daemon{
		Deployments: make(map[string]Deployment),
		//EventManager:    nil,
		quetzalOptions: opts,
	}
}

type Deployment struct {
	Name        string     `json:"name"`
	Path        string     `json:"-"`
	Description string     `json:"description,omitempty"`
	Hosts       []nix.Host `json:"hosts,omitempty"`
	Color       string     `json:"color"`
}

func (daemon *Daemon) LoadDeployments(deploymentsDir string) error {
	dirEntries, err := os.ReadDir(deploymentsDir)
	if err != nil {
		panic(1)
	}

	suffix := ".nix"

	for _, f := range dirEntries {
		if f.IsDir() {
			continue
		}

		if strings.HasSuffix(f.Name(), suffix) {
			p := path.Join(deploymentsDir, f.Name())

			daemon.quetzalOptions.Deployment = p

			fmt.Println(p)
			meta, hosts, err := cruft.GetHosts(daemon.quetzalOptions)

			if err != nil {
				panic(err.Error())
			}

			shortName := strings.TrimSuffix(f.Name(), suffix)
			daemon.Deployments[shortName] = Deployment{
				Name:        shortName,
				Path:        p,
				Description: meta.Description,
				Hosts:       hosts,
				Color:       meta.Color,
			}
		}
	}

	for name, f := range daemon.Deployments {
		fmt.Println(name)
		fmt.Println(f)
	}

	return nil
}
