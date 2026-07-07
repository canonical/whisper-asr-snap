package whisperlive

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os/exec"
	"strings"
	"time"
)

type TeeConfig struct {
	ChunkSize            int
	Rate                 int
	Channels             int
	RecordSeconds        int
	DisconnectGraceAfter time.Duration
}

func DefaultTeeConfig() TeeConfig {
	return TeeConfig{
		ChunkSize:            4096,
		Rate:                 16000,
		Channels:             1,
		RecordSeconds:        60000,
		DisconnectGraceAfter: 5 * time.Second,
	}
}

type TranscriptionTeeClient struct {
	Client *Client
	Config TeeConfig
}

func NewTranscriptionTeeClient(client *Client, cfg TeeConfig) (*TranscriptionTeeClient, error) {
	if client == nil {
		return nil, fmt.Errorf("client is required")
	}
	base := DefaultTeeConfig()
	if cfg.ChunkSize != 0 {
		base.ChunkSize = cfg.ChunkSize
	}
	if cfg.Rate != 0 {
		base.Rate = cfg.Rate
	}
	if cfg.Channels != 0 {
		base.Channels = cfg.Channels
	}
	if cfg.RecordSeconds != 0 {
		base.RecordSeconds = cfg.RecordSeconds
	}
	if cfg.DisconnectGraceAfter != 0 {
		base.DisconnectGraceAfter = cfg.DisconnectGraceAfter
	}

	return &TranscriptionTeeClient{Client: client, Config: base}, nil
}

func (t *TranscriptionTeeClient) WaitForServerReady(ctx context.Context) error {
	for {
		if t.Client.Recording() {
			return nil
		}
		if t.Client.Waiting() || t.Client.ServerError() {
			_ = t.CloseClient()
			return fmt.Errorf("server did not become ready")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(25 * time.Millisecond):
		}
	}
}

func (t *TranscriptionTeeClient) CloseClient() error {
	return t.Client.CloseWebSocket()
}

func (t *TranscriptionTeeClient) StreamPCM16LEReader(ctx context.Context, r io.Reader) error {
	reader := bufio.NewReader(r)
	chunk := make([]byte, t.Config.ChunkSize)

	for {
		if !t.Client.Recording() {
			break
		}

		n, err := io.ReadFull(reader, chunk)
		if n > 0 {
			// io.ReadFull leaves the final partial chunk in chunk[:n] on a short
			// read; send it so the tail of the audio is not dropped.
			audioFloat := bytesToFloat32PCM(chunk[:n])
			if sendErr := t.Client.SendPacketToServer(float32ToBytes(audioFloat)); sendErr != nil {
				return sendErr
			}
		}
		if err != nil {
			if errorsIsEOFLike(err) {
				break
			}
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	// Wait for the backend to finish transcribing the audio it has already
	// buffered before signaling end-of-audio. Sending END_OF_AUDIO too early
	// makes some backends stop transcription and close the socket, dropping the
	// final segment.
	_ = t.Client.WaitBeforeDisconnect(ctx)

	// Signal end-of-audio so backends that only flush on END_OF_AUDIO emit their
	// final segment, then wait briefly for that segment (or the server close)
	// before tearing the connection down.
	_ = t.Client.SendEndOfAudio()
	t.Client.WaitForServerClose(ctx, t.Config.DisconnectGraceAfter)
	return t.CloseClient()
}

func (t *TranscriptionTeeClient) ProcessRTSPStream(ctx context.Context, rtspURL string) error {
	return t.processFFmpegStream(ctx, rtspURL, "")
}

func (t *TranscriptionTeeClient) ProcessHLSStream(ctx context.Context, hlsURL string) error {
	return t.processFFmpegStream(ctx, hlsURL, "")
}

func (t *TranscriptionTeeClient) processFFmpegStream(ctx context.Context, inputURL, inputFormat string) error {
	if strings.TrimSpace(inputURL) == "" {
		return fmt.Errorf("input URL is required")
	}

	args := []string{"-hide_banner", "-loglevel", "error"}
	if inputFormat != "" {
		args = append(args, "-f", inputFormat)
	}
	args = append(args, "-i", inputURL, "-ac", "1", "-ar", "16000", "-f", "s16le", "pipe:1")

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("ffmpeg stdout pipe: %w", err)
	}
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start ffmpeg: %w", err)
	}

	streamErr := t.StreamPCM16LEReader(ctx, stdout)
	_ = cmd.Wait()

	if streamErr != nil {
		errOut, _ := io.ReadAll(stderr)
		if len(errOut) > 0 {
			return fmt.Errorf("ffmpeg stream failed: %v (%s)", streamErr, strings.TrimSpace(string(errOut)))
		}
		return streamErr
	}
	return nil
}

func errorsIsEOFLike(err error) bool {
	if err == nil {
		return false
	}
	if err == io.EOF || err == io.ErrUnexpectedEOF {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "eof")
}

func bytesToFloat32PCM(audioBytes []byte) []float32 {
	samples := make([]float32, 0, len(audioBytes)/2)
	for i := 0; i+1 < len(audioBytes); i += 2 {
		s := int16(binary.LittleEndian.Uint16(audioBytes[i:]))
		samples = append(samples, float32(s)/32768.0)
	}
	return samples
}

func float32ToBytes(samples []float32) []byte {
	buf := make([]byte, len(samples)*4)
	for i, sample := range samples {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(sample))
	}
	return buf
}
