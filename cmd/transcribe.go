package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	whisperlive "ubustt-proxy/backends/whisper_live"

	"github.com/spf13/cobra"
)

type transcribeCommand struct {
	host       string
	port       int
	audioPath  string
	model      string
	lang       string
	timeoutSec int
}

func NewTranscribeCmd() *cobra.Command {
	var cmd transcribeCommand

	cobraCmd := &cobra.Command{
		Use:               "transcribe",
		Short:             "Transcribe a local audio file with Whisper Live",
		Long:              "Connect to a Whisper Live websocket backend and transcribe a local audio file through ffmpeg streaming. Requires ffmpeg.",
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE:              cmd.run,
	}

	cobraCmd.Flags().StringVar(&cmd.host, "host", "127.0.0.1", "Whisper Live host")
	cobraCmd.Flags().IntVar(&cmd.port, "port", 9090, "Whisper Live port")
	cobraCmd.Flags().StringVar(&cmd.audioPath, "audio", "", "path to local audio file to transcribe")
	cobraCmd.Flags().StringVar(&cmd.model, "model", "small", "Whisper model name")
	cobraCmd.Flags().StringVar(&cmd.lang, "lang", "en", "source language code")
	cobraCmd.Flags().IntVar(&cmd.timeoutSec, "timeout-sec", 180, "command timeout in seconds")
	_ = cobraCmd.MarkFlagRequired("audio")

	return cobraCmd
}

func (cmd *transcribeCommand) run(cobraCmd *cobra.Command, _ []string) error {
	if _, err := os.Stat(cmd.audioPath); err != nil {
		return fmt.Errorf("audio file not accessible: %w", err)
	}
	if strings.TrimSpace(cmd.host) == "" || cmd.port <= 0 {
		return errors.New("host and port are required")
	}

	cfg := whisperlive.TranscriptionClientConfig{
		ClientConfig: whisperlive.ClientConfig{
			Host:             ptr(cmd.host),
			Port:             ptr(cmd.port),
			Lang:             ptr(cmd.lang),
			Model:            ptr(cmd.model),
			LogTranscription: ptr(true),
		},
	}

	transcriptionClient, err := whisperlive.NewTranscriptionClient(cfg)
	if err != nil {
		return fmt.Errorf("creating transcription client: %w", err)
	}
	defer func() {
		_ = transcriptionClient.Tee.CloseClient()
	}()

	timeout := time.Duration(cmd.timeoutSec) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	fmt.Fprintf(cobraCmd.OutOrStdout(), "connecting to whisper live at %s:%d\n", cmd.host, cmd.port)
	if err := transcriptionClient.Tee.WaitForServerReady(ctx); err != nil {
		return fmt.Errorf("waiting for server ready: %w", err)
	}

	fmt.Fprintf(cobraCmd.OutOrStdout(), "streaming audio file: %s\n", cmd.audioPath)
	if err := transcriptionClient.Tee.ProcessHLSStream(ctx, cmd.audioPath); err != nil {
		return fmt.Errorf("processing audio stream: %w", err)
	}

	segments := transcriptionClient.Client.Transcript()
	parts := make([]string, 0, len(segments)+1)
	for _, seg := range segments {
		text := strings.TrimSpace(seg.Text)
		if text != "" {
			parts = append(parts, text)
		}
	}

	// Append the trailing in-progress segment so the final line of speech is not
	// lost when the backend disconnects before marking it completed.
	if last := transcriptionClient.Client.LastSegment(); last != nil {
		text := strings.TrimSpace(last.Text)
		if text != "" && (len(parts) == 0 || parts[len(parts)-1] != text) {
			parts = append(parts, text)
		}
	}

	finalText := strings.TrimSpace(strings.Join(parts, " "))
	if finalText == "" {
		fmt.Fprintln(cobraCmd.OutOrStdout(), "transcription completed but no text was returned")
	} else {
		fmt.Fprintf(cobraCmd.OutOrStdout(), "final transcript:\n%s\n", finalText)
	}

	return nil
}

func ptr[T any](v T) *T {
	return &v
}
