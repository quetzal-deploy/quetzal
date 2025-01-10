package actions

import (
	"errors"
	"github.com/DBCDK/morph/cache"
	"github.com/DBCDK/morph/common"
	"github.com/DBCDK/morph/nix"
)

// FIXME: send this to some common thing
type Request struct {
	Description string            `json:"description"`
	Headers     map[string]string `json:"headers"`
	Host        *string           `json:"host"`
	InsecureSSL bool              `json:"insecureSSL"`
	Path        string            `json:"path"`
	Port        int               `json:"port"`
	Scheme      string            `json:"scheme"`
}

type LocalRequest struct {
	Request Request `json:"request"`
	Timeout int     `json:"timeout"`
}

type RemoteRequest struct {
	Request Request `json:"request"`
	Timeout int     `json:"timeout"`
}

func (_ LocalRequest) Name() string  { return "local-request" }
func (_ RemoteRequest) Name() string { return "remote-request" }

func (step LocalRequest) Run(mctx *common.MorphContext, allHosts map[string]nix.Host, cache_ *cache.Cache) error {
	return errors.New("not implemented: " + step.Name())
}

func (step RemoteRequest) Run(mctx *common.MorphContext, allHosts map[string]nix.Host, cache_ *cache.Cache) error {
	return errors.New("not implemented: " + step.Name())
}
