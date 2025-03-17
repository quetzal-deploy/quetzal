package events

import (
	"encoding/json"
	"fmt"
	"github.com/DBCDK/morph/common"
	"github.com/DBCDK/morph/cruft"
	"github.com/DBCDK/morph/nix"
	"github.com/labstack/echo/v4"
	"net/http"
	"os"
	"path"
	"strings"
)

var ()

type Deployment struct {
	Name        string     `json:"name"`
	Path        string     `json:"-"`
	Description string     `json:"description,omitempty"`
	Hosts       []nix.Host `json:"hosts,omitempty"`
	Color       string     `json:"color"`
}

func ServeHttp(opts *common.MorphOptions, port int, manager *Manager, deploymentsPath string) {
	dirEntries, err := os.ReadDir(deploymentsPath)
	if err != nil {
		panic(1)
	}

	deployments := make(map[string]Deployment) // make this a path or something instead

	suffix := ".nix"

	for _, f := range dirEntries {
		if f.IsDir() {
			continue
		}

		if strings.HasSuffix(f.Name(), suffix) {
			path := path.Join(deploymentsPath, f.Name())

			meta, hosts, err := cruft.GetHosts(opts)

			if err != nil {
				panic(err.Error())
			}

			shortName := strings.TrimSuffix(f.Name(), suffix)
			deployments[shortName] = Deployment{
				Name:        shortName,
				Path:        path,
				Description: meta.Description,
				Hosts:       hosts,
				Color:       meta.Color,
			}
		}
	}

	for name, f := range deployments {
		fmt.Println(name)
		fmt.Println(f)
	}

	e := echo.New()
	e.HideBanner = true

	e.GET("/events", func(c echo.Context) error {
		lastId := c.QueryParam("from")
		events, nextLastId := manager.GetEvents(lastId, 10)

		c.Response().Header().Set("Resume-From", nextLastId)
		c.Response().Header().Set("Resume-Link", fmt.Sprintf("/events?from=%s", nextLastId))

		for _, event := range events {
			eventJson, err := json.Marshal(event)

			if err != nil {
				c.Response().Write([]byte(err.Error()))
				break
			}

			c.Response().Write(eventJson)
			c.Response().Write([]byte("\n"))
		}

		return nil
	})

	e.GET("/deployments", func(c echo.Context) error {
		return c.JSONPretty(http.StatusOK, deployments, "  ")
	})

	e.GET("/deployments/:id", func(c echo.Context) error {

		deployment := deployments[c.Param("id")]

		//meta, hosts, _ := cruft.GetHosts(mctx, deployment.Path)
		//
		//deployment.Description = meta.Description
		//deployment.Hosts = hosts

		return c.JSONPretty(http.StatusOK, deployment, "  ")
	})

	e.Start(fmt.Sprintf("127.0.0.1:%d", port))
}
