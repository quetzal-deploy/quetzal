package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/quetzal-deploy/quetzal/internal/daemon"
	"github.com/quetzal-deploy/quetzal/internal/events"
	"github.com/quetzal-deploy/quetzal/internal/nix"
)

var (
	quetzalDaemon *daemon.Daemon
	eventManager  *events.Manager = nil
)

type Deployment struct {
	Name        string     `json:"name"`
	Path        string     `json:"-"`
	Description string     `json:"description,omitempty"`
	Hosts       []nix.Host `json:"hosts,omitempty"`
	Color       string     `json:"color"`
}

func Run(daemon *daemon.Daemon, port int, manager *events.Manager, deploymentsPath string) {
	quetzalDaemon = daemon
	eventManager = manager

	e := echo.New()
	e.HideBanner = true

	// FIXME: Configure logging
	// e.Logger =

	e.GET("/events", handlerGetEvents)

	if daemon != nil {
		e.GET("/deployments", handlerGetDeployments)
		e.GET("/deployments/:id", handlerGetDeploymentById)
	}

	e.Start(fmt.Sprintf("127.0.0.1:%d", port))
}

func handlerGetDeployments(c echo.Context) error {
	return c.JSONPretty(http.StatusOK, quetzalDaemon.Deployments, "  ")
}

func handlerGetDeploymentById(c echo.Context) error {

	deployment := quetzalDaemon.Deployments[c.Param("id")]

	//meta, hosts, _ := cruft.GetHosts(mctx, deployment.Path)
	//
	//deployment.Description = meta.Description
	//deployment.Hosts = hosts

	return c.JSONPretty(http.StatusOK, deployment, "  ")
}

func handlerGetEvents(c echo.Context) error {
	lastId := c.QueryParam("from")

	batchSize := 10
	batchSizeParamStr := c.QueryParam("limit")
	batchSizeParam, err := strconv.Atoi(batchSizeParamStr)
	if err == nil {
		batchSize = batchSizeParam
	}

	events, nextLastId := eventManager.GetEvents(lastId, batchSize)

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
}
