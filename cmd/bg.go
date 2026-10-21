package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/kothagpt/kotha/internal/config"
	"github.com/kothagpt/kotha/internal/llm/tools/shell"
	"github.com/spf13/cobra"
)

var bgWait bool

// loadConfigForCommand loads the config needed by the shell subsystem for
// standalone subcommands that bypass the root command's RunE setup.
func loadConfigForCommand(cmd *cobra.Command) error {
	debug, _ := cmd.Flags().GetBool("debug")
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %v", err)
	}
	if _, err := config.Load(cwd, debug); err != nil {
		return fmt.Errorf("failed to load config: %v", err)
	}
	return nil
}

var bgCmd = &cobra.Command{
	Use:   "bg [--wait] -- <command> [args...]",
	Short: "Run a shell command in the background",
	Long: `Run a shell command in the background and return a shell ID plus the
paths of its live output and error files. Use the shell ID with the
read_bash_output agent tool, or poll the output files directly.

Examples:
  kotha bg -- make test
  kotha bg --wait -- sleep 5 && echo done`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := loadConfigForCommand(cmd); err != nil {
			return err
		}
		if len(args) == 0 {
			return fmt.Errorf("no command provided; use: kotha bg -- <command>")
		}
		shellInst := shell.GetPersistentShell(".")
		defer shellInst.Close()

		bc, err := shellInst.StartBackground(cmd.Context(), strings.Join(args, " "), 0)
		if err != nil {
			return err
		}

		fmt.Printf("shell ID: %s\n", bc.ID)
		fmt.Printf("Output file: %s\n", bc.OutputFile)
		fmt.Printf("Error file: %s\n", bc.ErrorFile)

		if bgWait {
			res := bc.Wait()
			fmt.Printf("Exit code: %d\n", res.ExitCode)
			fmt.Printf("Output:\n%s\n", res.Stdout)
			if res.Stderr != "" {
				fmt.Printf("Errors:\n%s\n", res.Stderr)
			}
		}
		return nil
	},
}

var bgListCmd = &cobra.Command{
	Use:   "list",
	Short: "List background shell commands",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := loadConfigForCommand(cmd); err != nil {
			return err
		}
		shellInst := shell.GetPersistentShell(".")
		defer shellInst.Close()

		list := shellInst.ListBackgrounds()
		if len(list) == 0 {
			fmt.Println("no background commands")
			return nil
		}
		for _, bc := range list {
			status := "running"
			if bc.IsDone() {
				status = "done"
			}
			fmt.Printf("%s  %-7s  started %s  %s\n", bc.ID, status, bc.StartTime.Format(time.RFC822), bc.Command)
		}
		return nil
	},
}

var bgStopCmd = &cobra.Command{
	Use:   "stop <shell-id>",
	Short: "Stop a background shell command",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := loadConfigForCommand(cmd); err != nil {
			return err
		}
		shellInst := shell.GetPersistentShell(".")
		defer shellInst.Close()

		if shellInst.CancelBackground(args[0]) {
			fmt.Println("cancelled", args[0])
		} else {
			fmt.Println("not found or already finished:", args[0])
		}
		return nil
	},
}

func init() {
	bgCmd.Flags().BoolVar(&bgWait, "wait", false, "wait for the command to complete and print its output")
	bgCmd.AddCommand(bgListCmd, bgStopCmd)
	rootCmd.AddCommand(bgCmd)
}
