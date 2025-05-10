package steps

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/quetzal-deploy/quetzal/internal/cache"
	"github.com/quetzal-deploy/quetzal/internal/common"
	"github.com/quetzal-deploy/quetzal/internal/logging"
	"github.com/quetzal-deploy/quetzal/internal/nix"
	"github.com/quetzal-deploy/quetzal/internal/ssh"
)

type IsOnline struct {
	Host string `json:"host"`
}

type Reboot struct {
	Host string `json:"host"`
}

func (_ IsOnline) Name() string { return "is-online" }
func (_ Reboot) Name() string   { return "reboot" }

func (action *IsOnline) MarshalJSONx(step Step) ([]byte, error) {
	return json.Marshal(struct {
		StepAlias
		IsOnline
	}{
		StepAlias: StepAlias(step),
		IsOnline:  *action,
	})
}

func (action *Reboot) MarshalJSONx(step Step) ([]byte, error) {
	return json.Marshal(struct {
		StepAlias
		Reboot
	}{
		StepAlias: StepAlias(step),
		Reboot:    *action,
	})
}

func (action *IsOnline) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, action)
}

func (action *Reboot) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, action)
}

func (action IsOnline) Run(ctx context.Context, opts *common.QuetzalOptions, allHosts map[string]nix.Host, cache_ *cache.LockedMap[string]) error {
	sshContext := ssh.CreateSSHContext(opts)

	host, ok := allHosts[action.Host]
	if !ok {
		return errors.New(fmt.Sprintf("host '%s' not in deployment", action.Host))
	}

	cmd, err := sshContext.CmdContext(ctx, &host, "/bin/sh", "-c", "true")
	if err != nil {
		return err
	}

	logging.LogCmd(action.Host, cmd)

	err = cmd.Run()

	return err
}
func (action Reboot) Run(ctx context.Context, opts *common.QuetzalOptions, allHosts map[string]nix.Host, cache_ *cache.LockedMap[string]) error {
	sshContext := ssh.CreateSSHContext(opts)

	host, exists := allHosts[action.Host]
	if !exists {
		return errors.New("unknown host: " + action.Host)
	}

	err := host.Reboot(sshContext)

	return err
}
