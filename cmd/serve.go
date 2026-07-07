package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ubustt-proxy/backends"
	whisperlive "ubustt-proxy/backends/whisper_live"
	"ubustt-proxy/ubustt/server"

	"github.com/spf13/cobra"
)

type serveCommand struct {
	host       string
	port       int
	unixSocket string

	backendHost  string
	backendPort  int
	backendModel string
	backendLang  string
}

func NewServeCmd() *cobra.Command {
	var cmd serveCommand

	cobraCmd := &cobra.Command{
		Use:               "serve",
		Short:             "Start the websocket server",
		Long:              "Start a UbuSTT websocket server that proxies each connection to a Whisper Live transcription backend.",
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE:              cmd.run,
	}

	cobraCmd.Flags().StringVar(&cmd.host, "host", "127.0.0.1", "host interface to bind")
	cobraCmd.Flags().IntVar(&cmd.port, "port", 8080, "port to listen on")
	cobraCmd.Flags().StringVar(&cmd.unixSocket, "unix-socket", "", "path to a Unix domain socket to bind (overrides --host/--port)")

	cobraCmd.Flags().StringVar(&cmd.backendHost, "whisper-host", "127.0.0.1", "Whisper Live backend host")
	cobraCmd.Flags().IntVar(&cmd.backendPort, "whisper-port", 9090, "Whisper Live backend port")
	cobraCmd.Flags().StringVar(&cmd.backendModel, "whisper-model", "small", "Whisper model name")
	cobraCmd.Flags().StringVar(&cmd.backendLang, "whisper-lang", "en", "source language code")

	return cobraCmd
}

func (cmd *serveCommand) run(cobraCmd *cobra.Command, _ []string) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	srv := server.NewWebSocketServer(cmd.host, cmd.port, cmd.unixSocket)
	srv.SetBackend(
		backends.SessionConfig{
			Model: cmd.backendModel,
			Lang:  cmd.backendLang,
		},
		func(ctx context.Context, onDelta func(string), onCommit func(string)) (backends.Backend, error) {
			return whisperlive.Dial(ctx, whisperlive.Config{
				Host:     cmd.backendHost,
				Port:     cmd.backendPort,
				Model:    cmd.backendModel,
				Lang:     cmd.backendLang,
				OnDelta:  onDelta,
				OnCommit: onCommit,
			})
		},
	)
	fmt.Fprintf(cobraCmd.OutOrStdout(), "starting websocket server on %s\n", srv.Address())

	startErr := make(chan error, 1)
	go func() { startErr <- srv.Start() }()

	select {
	case err := <-startErr:
		return err
	case <-ctx.Done():
		stop() // restore default SIGINT behaviour so a second Ctrl+C kills immediately
		fmt.Fprintf(cobraCmd.OutOrStdout(), "\nshutting down...\n")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Stop(shutdownCtx)
		return <-startErr
	}
}
