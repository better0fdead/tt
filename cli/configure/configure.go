package configure

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tarantool/tt/cli/context"
	"github.com/tarantool/tt/cli/modules"
	"github.com/tarantool/tt/cli/util"
)

const (
	configName        = "tarantool.yaml"
	cliExecutableName = "tt"
)

var (
	// Path to default tarantool.yaml configuration file.
	// Defined at build time, see magefile.
	defaultConfigPath string
)

// Cli performs initial CLI configuration.
func Cli(ctx *context.Ctx) error {
	if ctx.Cli.ConfigPath != "" {
		if _, err := os.Stat(ctx.Cli.ConfigPath); err != nil {
			return fmt.Errorf("Specified path to the configuration file is invalid: %s", err)
		}
	}

	switch {
	case ctx.Cli.IsSystem:
		return configureSystemCli(ctx)
	case ctx.Cli.LocalLaunchDir != "":
		return configureLocalCli(ctx, ctx.Cli.LocalLaunchDir)
	}

	// No flags specified.
	return configureDefaultCli(ctx)
}

// ExternalCmd configures external commands.
func ExternalCmd(rootCmd *cobra.Command, ctx *context.Ctx, modulesInfo *modules.ModulesInfo, args []string) {
	configureExistsCmd(rootCmd, modulesInfo)
	configureNonExistentCmd(rootCmd, ctx, modulesInfo, args)
}

// configureExistsCmd configures an external commands
// that have internal implemetation.
func configureExistsCmd(rootCmd *cobra.Command, modulesInfo *modules.ModulesInfo) {
	for _, cmd := range rootCmd.Commands() {
		if module, ok := (*modulesInfo)[cmd.Name()]; ok {
			if !module.IsInternal {
				cmd.DisableFlagParsing = true
			}
		}
	}
}

// configureNonExistentCmd configures an external command that
// has no internal implementation within the Tarantool CLI.
func configureNonExistentCmd(rootCmd *cobra.Command, ctx *context.Ctx, modulesInfo *modules.ModulesInfo, args []string) {
	// Since the user can pass flags, to determine the name of
	// an external command we have to take the first non-flag argument.
	externalCmd := args[0]
	for _, name := range args {
		if !strings.HasPrefix(name, "-") && name != "help" {
			externalCmd = name
			break
		}
	}

	// We avoid overwriting existing commands - we should add a command only
	// if it doesn't have an internal implementation in Tarantool CLI.
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == externalCmd {
			return
		}
	}

	helpCmd := util.GetHelpCommand(rootCmd)
	if module, ok := (*modulesInfo)[externalCmd]; ok {
		if !module.IsInternal {
			rootCmd.AddCommand(newExternalCommand(ctx, modulesInfo, externalCmd, nil))
			helpCmd.AddCommand(newExternalCommand(ctx, modulesInfo, externalCmd, []string{"--help"}))
		}
	}
}

// newExternalCommand returns a pointer to a new external
// command that will call modules.RunCmd.
func newExternalCommand(ctx *context.Ctx, modulesInfo *modules.ModulesInfo, cmdName string, addArgs []string) *cobra.Command {
	cmd := &cobra.Command{
		Use: cmdName,
		Run: func(cmd *cobra.Command, args []string) {
			if addArgs != nil {
				args = append(args, addArgs...)
			}

			ctx.Cli.ForceInternal = false
			if err := modules.RunCmd(ctx, cmdName, modulesInfo, nil, args); err != nil {
				log.Fatalf(err.Error())
			}
		},
	}

	cmd.DisableFlagParsing = true
	return cmd
}

// comfigureLocalCli configures Tarantool CLI if the launch is local.
func configureLocalCli(ctx *context.Ctx, launchDir string) error {
	// If tt launch is local: we chdir to a local directory, check for tt
	// and Tarantool binaries. If tt binary exists, then exec it.
	// If Tarantool binary is found, use it further, instead of what
	// is specified in the PATH.

	launchDir, err := filepath.Abs(launchDir)
	if err != nil {
		return fmt.Errorf(`Failed to get absolute path to local directory: %s`, err)
	}

	if err := os.Chdir(launchDir); err != nil {
		return fmt.Errorf(`Failed to change working directory: %s`, err)
	}

	if ctx.Cli.ConfigPath == "" {
		ctx.Cli.ConfigPath = filepath.Join(launchDir, configName)
		// TODO: Add warning messages, discussion what if the file
		// exists, but access is denied, etc.
		if _, err := os.Stat(ctx.Cli.ConfigPath); err != nil {
			if !os.IsNotExist(err) {
				return fmt.Errorf("Failed to get access to configuration file: %s", err)
			}

			var err error
			if ctx.Cli.ConfigPath, err = getConfigPath(configName); err != nil {
				return fmt.Errorf("Failed to get Tarantool CLI config: %s", err)
			}

			if ctx.Cli.ConfigPath == "" {
				ctx.Cli.ConfigPath = filepath.Join(defaultConfigPath, configName)
			}
		}
	}

	// Detect local tarantool.
	localTarantool, err := util.JoinAbspath(launchDir, "tarantool")
	if err != nil {
		return err
	}

	if _, err := os.Stat(localTarantool); err == nil {
		if _, err := exec.LookPath(localTarantool); err != nil {
			return fmt.Errorf(
				`Found Tarantool binary in local directory "%s" isn't executable: %s`, launchDir, err)
		}

		ctx.Cli.TarantoolExecutable = localTarantool
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("Failed to get access to Tarantool binary file: %s", err)
	}

	// Detect local tt.
	localCli, err := util.JoinAbspath(ctx.Cli.LocalLaunchDir, cliExecutableName)
	if err != nil {
		return err
	}

	// This should save us from exec looping.
	if localCli != os.Args[0] {
		if _, err := os.Stat(localCli); err == nil {
			if _, err := exec.LookPath(localCli); err != nil {
				return fmt.Errorf(
					`Found tt binary in local directory "%s" isn't executable: %s`, launchDir, err)
			}

			rc := modules.RunExec(localCli, os.Args[1:])
			os.Exit(rc)
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("Failed to get access to tt binary file: %s", err)
		}
	}

	return nil
}

// configureSystemCli configures Tarantool CLI if the launch is system.
func configureSystemCli(ctx *context.Ctx) error {
	// If tt launch is system: the only thing we do is look for tarantool.yaml
	// config in the system directory (as opposed to running it locally).
	if ctx.Cli.ConfigPath == "" {
		ctx.Cli.ConfigPath = filepath.Join(defaultConfigPath, configName)
	}

	return nil
}

// configureDefaultCLI configures Tarantool CLI if the launch was without flags (-S or -L).
func configureDefaultCli(ctx *context.Ctx) error {
	// If neither the local start nor the system flag is specified,
	// we ourselves determine what kind of launch it is.

	if ctx.Cli.ConfigPath == "" {
		// We start looking for config in the current directory, going down to root directory.
		// If the config is found, we assume that it is a local launch in this directory.
		// If the config is not found, then we take it from the standard place (/etc/tarantool).

		var err error
		if ctx.Cli.ConfigPath, err = getConfigPath(configName); err != nil {
			return fmt.Errorf("Failed to get Tarantool CLI config: %s", err)
		}
	}

	if ctx.Cli.ConfigPath != "" {
		return configureLocalCli(ctx, filepath.Dir(ctx.Cli.ConfigPath))
	}

	return configureSystemCli(ctx)
}

// getConfigPath looks for the path to the tarantool.yaml configuration file,
// looking through all directories from the current one to the root.
// This search pattern is chosen for the convenience of the user.
func getConfigPath(configName string) (string, error) {
	curDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("Failed to detect current directory: %s", err)
	}

	for curDir != "/" {
		configPath := filepath.Join(curDir, configName)
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}

		curDir = filepath.Dir(curDir)
	}

	return "", nil
}