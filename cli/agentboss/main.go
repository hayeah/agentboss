package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "dev"

func main() {
	app, cleanup, err := InitializeApp()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()

	if err := app.rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// App holds the wired dependencies and root command.
type App struct {
	rootCmd *cobra.Command
}

// NewApp creates the App with all subcommands.
func NewApp(boss *Boss) *App {
	rootCmd := &cobra.Command{
		Use:     "agentboss",
		Short:   "Generic interactive CLI supervisor via tmux",
		Version: version,
	}

	rootCmd.AddCommand(
		newRunCmd(boss),
		newOutputCmd(boss),
		newSendCmd(boss),
		newExpectCmd(boss),
		newStatusCmd(boss),
		newWaitCmd(boss),
		newLsCmd(boss),
		newAttachCmd(boss),
	)

	return &App{rootCmd: rootCmd}
}
