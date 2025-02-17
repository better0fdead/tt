package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/apex/log"
	"github.com/spf13/cobra"
	"github.com/tarantool/tt/cli/cmdcontext"
	"github.com/tarantool/tt/cli/modules"
	"github.com/tarantool/tt/cli/process_utils"
	"github.com/tarantool/tt/cli/running"
	"github.com/tarantool/tt/cli/util"
)

var forceRemove bool

// NewCleanCmd creates clean command.
func NewCleanCmd() *cobra.Command {
	var cleanCmd = &cobra.Command{
		Use:   "clean [INSTANCE_NAME]",
		Short: "Clean instance(s) files",
		Run: func(cmd *cobra.Command, args []string) {
			err := modules.RunCmd(&cmdCtx, cmd.CommandPath(), &modulesInfo, internalCleanModule,
				args)
			handleCmdErr(cmd, err)
		},
	}

	cleanCmd.Flags().BoolVarP(&forceRemove, "force", "f", false, "do not ask for confirmation")

	return cleanCmd
}

func collectFiles(list []string, dirname string) ([]string, error) {
	err := filepath.Walk(dirname,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if !info.IsDir() {
				list = append(list, path)
			}
			return nil
		})
	if err != nil {
		return nil, err
	}

	return list, nil
}

func clean(run *running.InstanceCtx) error {
	removeList := []string{}
	confirm := false
	var err error

	for _, dir := range [...]string{run.LogDir, run.WalDir, run.VinylDir, run.MemtxDir} {
		removeList, err = collectFiles(removeList, dir)
		if err != nil {
			return err
		}
	}

	if len(removeList) == 0 {
		log.Infof("Already cleaned.\n")
		return nil
	}

	log.Infof("List of files to delete:\n")
	for _, file := range removeList {
		log.Infof("%s", file)
	}

	if !forceRemove {
		confirm, err = util.AskConfirm(os.Stdin, "\nConfirm")
		if err != nil {
			return err
		}
	}

	if confirm || forceRemove {
		for _, file := range removeList {
			err = os.Remove(file)
			if err != nil {
				return err
			}
		}

		return nil
	}

	return fmt.Errorf("canceled by user")
}

// internalCleanModule is a default clean module.
func internalCleanModule(cmdCtx *cmdcontext.CmdCtx, args []string) error {
	if !isConfigExist(cmdCtx) {
		return errNoConfig
	}

	var runningCtx running.RunningCtx
	if err := running.FillCtx(cliOpts, cmdCtx, &runningCtx, args); err != nil {
		return err
	}

	for _, run := range runningCtx.Instances {
		status := running.Status(&run)
		if status.Code == process_utils.ProcessStoppedCode {
			var statusMsg string

			err := clean(&run)
			if err != nil {
				statusMsg = "[ERR] " + err.Error()
			} else {
				statusMsg = "[OK]"
			}

			log.Infof("%s: cleaning...\t%s", run.InstName, statusMsg)
		} else {
			log.Infof("instance `%s` must be stopped", run.InstName)
		}
	}

	return nil
}
