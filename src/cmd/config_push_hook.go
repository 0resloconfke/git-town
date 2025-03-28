package cmd

import (
	"fmt"

	"github.com/git-town/git-town/v8/src/cli"
	"github.com/git-town/git-town/v8/src/config"
	"github.com/git-town/git-town/v8/src/execute"
	"github.com/git-town/git-town/v8/src/flags"
	"github.com/git-town/git-town/v8/src/git"
	"github.com/spf13/cobra"
)

const pushHookDesc = "Configures whether Git Town should run Git's pre-push hook."

const pushHookHelp = `
Enabled by default. When disabled, Git Town prevents Git's pre-push hook from running.`

func pushHookCommand() *cobra.Command {
	addDebugFlag, readDebugFlag := flags.Debug()
	addGlobalFlag, readGlobalFlag := flags.Bool("global", "g", "If set, reads or updates the push hook flag for all repos on this machine")
	cmd := cobra.Command{
		Use:   "push-hook [--global] [(yes | no)]",
		Args:  cobra.MaximumNArgs(1),
		Short: pushHookDesc,
		Long:  long(pushHookDesc, pushHookHelp),
		RunE: func(cmd *cobra.Command, args []string) error {
			return pushHook(args, readGlobalFlag(cmd), readDebugFlag(cmd))
		},
	}
	addDebugFlag(&cmd)
	addGlobalFlag(&cmd)
	return &cmd
}

func pushHook(args []string, global, debug bool) error {
	run, exit, err := execute.LoadProdRunner(execute.LoadArgs{
		OmitBranchNames:       true,
		Debug:                 debug,
		DryRun:                false,
		HandleUnfinishedState: false,
		ValidateGitversion:    true,
		ValidateIsRepository:  true,
	})
	if err != nil || exit {
		return err
	}
	if len(args) > 0 {
		return setPushHook(args[0], global, &run)
	}
	return printPushHook(global, &run)
}

func printPushHook(globalFlag bool, run *git.ProdRunner) error {
	var setting bool
	var err error
	if globalFlag {
		setting, err = run.Config.PushHookGlobal()
	} else {
		setting, err = run.Config.PushHook()
	}
	if err != nil {
		return err
	}
	cli.Println(cli.FormatBool(setting))
	return nil
}

func setPushHook(text string, global bool, run *git.ProdRunner) error {
	value, err := config.ParseBool(text)
	if err != nil {
		return fmt.Errorf(`invalid argument: %q. Please provide either "yes" or "no"`, text)
	}
	if global {
		return run.Config.SetPushHookGlobally(value)
	}
	return run.Config.SetPushHookLocally(value)
}
