package cmd

import (
	"fmt"

	"ubustt-proxy/ubustt/server"

	"github.com/spf13/cobra"
)

type serveCommand struct {
	host       string
	port       int
	unixSocket string
}

func NewServeCmd() *cobra.Command {
	var cmd serveCommand

	cobraCmd := &cobra.Command{
		Use:               "serve",
		Short:             "Start the websocket server",
		Long:              "Start a websocket echo server and expose it on the configured host/port or on a Unix socket.",
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE:              cmd.run,
	}

	cobraCmd.Flags().StringVar(&cmd.host, "host", "127.0.0.1", "host interface to bind")
	cobraCmd.Flags().IntVar(&cmd.port, "port", 8080, "port to listen on")
	cobraCmd.Flags().StringVar(&cmd.unixSocket, "unix-socket", "", "path to a Unix domain socket to bind (overrides --host/--port)")

	return cobraCmd
}

func (cmd *serveCommand) run(cobraCmd *cobra.Command, _ []string) error {
	srv := server.NewWebSocketServer(cmd.host, cmd.port, cmd.unixSocket)
	fmt.Fprintf(cobraCmd.OutOrStdout(), "starting websocket server on %s\n", srv.Address())
	return srv.Start()
}
