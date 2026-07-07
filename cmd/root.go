package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	cobra.EnableCommandSorting = false

	rootCmd := &cobra.Command{
		Use:           "ubustt-proxy",
		Short:         "A small websocket server for proxying text streams",
		Long:          "Start and manage the ubustt websocket server.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	rootCmd.CompletionOptions.HiddenDefaultCmd = true

	rootCmd.AddCommand(NewServeCmd())
	rootCmd.AddCommand(NewTranscribeCmd())
	return rootCmd
}

func Execute() error {
	if err := NewRootCmd().Execute(); err != nil {
		return fmt.Errorf("executing root command: %w", err)
	}
	return nil
}
