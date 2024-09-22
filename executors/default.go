package executors

import (
	"fmt"
	"path"
	"path/filepath"

	"github.com/DBCDK/morph/common"
	"github.com/DBCDK/morph/cruft"
	"github.com/DBCDK/morph/nix"
	"github.com/DBCDK/morph/planner"
	"github.com/DBCDK/morph/ssh"
)

var (
	cache     = make(map[string]string)
	cacheChan = make(chan StepData)
)

type StepData struct {
	Key   string
	Value string
}

func cacheWriter() {
	for update := range cacheChan {
		fmt.Printf("cache update: %s = %s\n", update.Key, update.Value)
		cache[update.Key] = update.Value
	}
}

type DefaultPlanExecutor struct {
	Hosts        map[string]nix.Host
	MorphContext *common.MorphContext
	SSHContext   *ssh.SSHContext
	NixContext   *nix.NixContext
}

func (ex DefaultPlanExecutor) Init() error {
	go cacheWriter()

	return nil
}

func (ex DefaultPlanExecutor) TearDown() error {

	return nil
}

func (ex DefaultPlanExecutor) GetHosts() map[string]nix.Host {
	return ex.Hosts
}

func (ex DefaultPlanExecutor) GetMorphContext() *common.MorphContext {
	return ex.MorphContext
}

func (ex DefaultPlanExecutor) GetSSHContext() *ssh.SSHContext {
	return ex.SSHContext
}

func (ex DefaultPlanExecutor) GetNixContext() *nix.NixContext {
	return ex.NixContext
}

func (executor DefaultPlanExecutor) Build(step planner.Step) error {
	hostsByName := step.Options["hosts"].([]string)

	nixHosts := make([]nix.Host, 0)

	fmt.Println("Building hosts:")
	for _, hostByName := range hostsByName {
		fmt.Printf("- %s\n", hostByName)
		nixHosts = append(nixHosts, executor.GetHosts()[hostByName])
	}

	resultPath, err := cruft.ExecBuild(executor.MorphContext, nixHosts)
	if err != nil {
		return err
	}

	fmt.Println(resultPath)

	for _, host := range nixHosts {
		hostPathSymlink := path.Join(resultPath, host.Name)
		hostPath, err := filepath.EvalSymlinks(hostPathSymlink)
		if err != nil {
			return err
		}

		fmt.Println(hostPathSymlink)
		fmt.Println(hostPath)

		cacheChan <- StepData{Key: "closure:" + host.Name, Value: hostPath}

		// store hostPath to be fetched by other steps
	}

	return err
}

func (executor DefaultPlanExecutor) Push(step planner.Step) error {
	cacheKey := "closure:" + step.Host.Name
	fmt.Println("cache key: " + cacheKey)
	closure := cache[cacheKey]

	fmt.Printf("Pushing %s to %s\n", closure, step.Host.TargetHost)

	sshContext := executor.GetSSHContext()
	err := nix.Push(sshContext, *step.Host, closure)

	return err
}

func (executor DefaultPlanExecutor) DeploySwitch(step planner.Step) error {

	return nil
}

func (executor DefaultPlanExecutor) DeployBoot(step planner.Step) error {

	return nil
}

func (executor DefaultPlanExecutor) DeployDryActivate(step planner.Step) error {

	return nil
}

func (executor DefaultPlanExecutor) DeployTest(step planner.Step) error {

	return nil
}

func (executor DefaultPlanExecutor) Reboot(step planner.Step) error {

	return nil
}

func (executor DefaultPlanExecutor) CommandCheckLocal(step planner.Step) error {

	return nil
}

func (executor DefaultPlanExecutor) CommandCheckRemote(step planner.Step) error {

	return nil
}

func (executor DefaultPlanExecutor) HttpCheckLocal(step planner.Step) error {

	return nil
}

func (executor DefaultPlanExecutor) HttpCheckRemote(step planner.Step) error {

	return nil
}
