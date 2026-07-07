package cmd

import (
	"bufio"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"ubustt-proxy/ubustt/messages"

	"github.com/gorilla/websocket"
	"github.com/spf13/cobra"
)

type clientCommand struct {
	url            string
	audioPath      string
	sampleRate     int
	chunkBytes     int
	timeoutSec     int
	realtimeFactor float64

	ctx                *context.Context
	streamingCompleted bool
}

// NewClientCmd builds a small UbuSTT test client. It connects to a running
// UbuSTT websocket server, streams a local audio file as
// input_audio_buffer.append events paced in real time, then commits and prints
// the transcription events as they arrive.
func NewClientCmd() *cobra.Command {
	var cmd clientCommand

	cobraCmd := &cobra.Command{
		Use:               "client",
		Short:             "Stream an audio file to a UbuSTT server and print transcriptions",
		Long:              "Connect to a running UbuSTT websocket server, stream a local audio file (decoded with ffmpeg) as input_audio_buffer.append events, commit, and print transcription deltas/completions. Requires ffmpeg.",
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE:              cmd.run,
	}

	cobraCmd.Flags().StringVar(&cmd.url, "url", "ws://127.0.0.1:8080/ws", "UbuSTT websocket URL")
	cobraCmd.Flags().StringVar(&cmd.audioPath, "audio", "data/samples/jfk.flac", "path to local audio file to stream")
	cobraCmd.Flags().IntVar(&cmd.sampleRate, "rate", 16000, "sample rate to resample the audio to")
	cobraCmd.Flags().IntVar(&cmd.chunkBytes, "chunk-bytes", 4096, "PCM16 bytes per append event")
	cobraCmd.Flags().IntVar(&cmd.timeoutSec, "timeout-sec", 180, "overall command timeout in seconds")
	cobraCmd.Flags().Float64Var(&cmd.realtimeFactor, "realtime-factor", 1.0, "factor to adjust real-time pacing of audio streaming")

	return cobraCmd
}

func (cmd *clientCommand) run(cobraCmd *cobra.Command, _ []string) error {
	if _, err := os.Stat(cmd.audioPath); err != nil {
		return fmt.Errorf("audio file not accessible: %w", err)
	}
	if strings.TrimSpace(cmd.url) == "" {
		return errors.New("url is required")
	}
	if cmd.realtimeFactor < 1 {
		return errors.New("realtime-factor must be >= 1")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.timeoutSec)*time.Second)
	defer cancel()
	cmd.ctx = &ctx

	out := cobraCmd.OutOrStdout()

	fmt.Fprintf(out, "connecting to %s\n", cmd.url)
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, cmd.url, nil)
	if err != nil {
		return fmt.Errorf("dial ubustt server: %w", err)
	}
	defer conn.Close()

	// Read events in the background and print them until the connection closes.
	done := make(chan struct{})
	go func() {
		defer close(done)
		cmd.readLoop(out, conn)
	}()

	// Wait for the server's session.created before streaming audio.
	fmt.Fprintln(out, "waiting for session.created")

	// The server flushes the final segment and closes the connection on commit,
	// which ends the read loop. Bound the wait by the context.
	select {
	case <-done:
	case <-ctx.Done():
		return ctx.Err()
	}
	return nil
}

func (cmd *clientCommand) sendAudio(out io.Writer, conn *websocket.Conn) error {
	// Stream the audio, then commit to signal end of input.
	if err := cmd.stream(out, conn); err != nil {
		return fmt.Errorf("streaming audio: %w", err)
	}

	commit, err := messages.ToJson(&messages.InputAudioBufferCommit{})
	if err != nil {
		return fmt.Errorf("encoding commit: %w", err)
	}
	if err := conn.WriteMessage(websocket.TextMessage, commit); err != nil {
		return fmt.Errorf("sending commit: %w", err)
	}

	fmt.Fprintf(out, "audio streaming complete\n")
	cmd.streamingCompleted = true
	return nil
}

// stream decodes the audio file to PCM16 mono with ffmpeg and sends it as
// base64 input_audio_buffer.append events, paced at real time.
func (cmd *clientCommand) stream(out io.Writer, conn *websocket.Conn) error {
	ff := exec.CommandContext(*cmd.ctx, "ffmpeg",
		"-hide_banner", "-loglevel", "error",
		"-i", cmd.audioPath,
		"-ac", "1",
		"-ar", fmt.Sprintf("%d", cmd.sampleRate),
		"-f", "s16le",
		"pipe:1",
	)
	stdout, err := ff.StdoutPipe()
	if err != nil {
		return fmt.Errorf("ffmpeg stdout pipe: %w", err)
	}
	stderr := &strings.Builder{}
	ff.Stderr = stderr

	if err := ff.Start(); err != nil {
		return fmt.Errorf("start ffmpeg: %w", err)
	}

	fmt.Fprintf(out, "streaming %s\n", cmd.audioPath)
	reader := bufio.NewReader(stdout)
	chunk := make([]byte, cmd.chunkBytes)

	for {
		n, readErr := io.ReadFull(reader, chunk)
		if n > 0 {
			payload := chunk[:n]
			msg, err := messages.ToJson(&messages.InputAudioBufferAppend{
				Audio: base64.StdEncoding.EncodeToString(payload),
			})
			if err != nil {
				return fmt.Errorf("encoding append: %w", err)
			}
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return fmt.Errorf("sending append: %w", err)
			}
			// Pace at real time: s16le mono => 2 bytes per sample.
			samples := n / 2
			pace := time.Duration(float64(samples) / float64(cmd.sampleRate) * float64(time.Second) / cmd.realtimeFactor)
			select {
			case <-(*cmd.ctx).Done():
				return (*cmd.ctx).Err()
			case <-time.After(pace):
			}
		}
		if readErr != nil {
			if readErr == io.EOF || readErr == io.ErrUnexpectedEOF {
				break
			}
			return readErr
		}
	}

	if err := ff.Wait(); err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return fmt.Errorf("ffmpeg failed: %w (%s)", err, msg)
		}
		return fmt.Errorf("ffmpeg failed: %w", err)
	}
	return nil
}

// readLoop prints inbound server events until the connection closes.
func (cmd *clientCommand) readLoop(out io.Writer, conn *websocket.Conn) {
	for {
		_, payload, err := conn.ReadMessage()
		if err != nil {
			return
		}

		msg, err := messages.FromJson(payload)
		if err != nil {
			fmt.Fprintf(out, "[unparsed] %s\n", string(payload))
			continue
		}

		switch m := msg.(type) {
		case *messages.SessionCreated:
			fmt.Fprintf(out, "[session.created] model=%s lang=%s rate=%d\n",
				m.Session.Audio.Input.Transcription.Model,
				m.Session.Audio.Input.Transcription.Language,
				m.Session.Audio.Input.Format.Rate)

		case *messages.SessionUpdated:
			fmt.Fprintf(out, "[session.updated]: %v\n", m.Session)
		case *messages.ConversationItemInputAudioTranscriptionDelta:
			fmt.Fprintf(out, "[delta] %s\n", m.Delta)
		case *messages.ModelLoaded:
			fmt.Fprintf(out, "[model.loaded]\n")
			go func() {
				if err := cmd.sendAudio(out, conn); err != nil {
					fmt.Fprintf(out, "error sending audio: %v\n", err)
				}
				fmt.Fprintln(out, "audio committed, waiting for final transcription")
			}()
		case *messages.ModelUnloaded:
			fmt.Fprintf(out, "[model.unloaded]\n")
		case *messages.ConversationItemInputAudioTranscriptionCompleted:
			fmt.Fprintf(out, "[completed] %s\n", m.Transcript)
			if cmd.streamingCompleted {
				fmt.Fprintln(out, "transcription completed, closing connection")
				_ = conn.Close()
			}
		case *messages.Error:
			fmt.Fprintf(out, "[error] %s/%s: %s\n", m.Error.Type, m.Error.Code, m.Error.Message)
		default:
			fmt.Fprintf(out, "[%T]\n", m)
		}
	}
}
