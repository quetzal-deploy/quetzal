package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func Execute(version string, assets string) {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use: "quetzal",
	Long: `Quetzal is a deployment tool for managing NixOS hosts.

<Quetzal> I've been waiting for a long time. It's time to hatch.`,
}

var plansCmd = &cobra.Command{
	Use:   "plans <deployment-file> ( list | show <plan> | run <plan> )",
	Short: "List plans in a deployment, show individual plans rendered as JSON and run them.",
	Args:  cobra.MinimumNArgs(2),
	Example: `
quetzal plans <deployment-file> list
quetzal plans <deployment-file> show build
quetzal plans <deployment-file> run switch`,

	RunE: func(cmd *cobra.Command, args []string) error {
		deployment := args[0]
		action := args[1]

		switch action {
		case "list":
			if len(args) != 2 {
				return fmt.Errorf("wrong numbner of positional arguments")
			}
			return listPlans(deployment)
		case "show":
			if len(args) != 3 {
				return errors.New("wrong numbner of positional arguments")
			}
			plan := args[2]
			return showPlan(deployment, plan)
		case "run":
			if len(args) != 3 {
				return errors.New("wrong numbner of positional arguments")
			}
			plan := args[2]
			return runPlan(deployment, plan)
		default:
			return fmt.Errorf("unknown action %q. expected one of: list, show, run", action)
		}
	},
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the same named plan from multiple deployment files.",
	Args:  cobra.MinimumNArgs(2),
	Example: `
quetzal run build <deployment-file>
quetzal run build <deployment-file> <deployment-file>`,

	RunE: func(cmd *cobra.Command, args []string) error {
		plan := args[0]
		deployments := args[1:]
		return runPlans(plan, deployments)
	},
}

func init() {
	rootCmd.AddCommand(plansCmd)
	rootCmd.AddCommand(runCmd)
}

func listPlans(file string) error {
	// TODO
	return nil
}

func showPlan(file, plan string) error {
	// TODO
	return nil
}

func runPlan(deployment, plan string) error {
	// TODO
	return nil
}

func runPlans(plan string, deployments []string) error {
	for _, deployment := range deployments {
		err := runPlan(plan, deployment)
		if err != nil {
			return err
		}
	}

	return nil
}
