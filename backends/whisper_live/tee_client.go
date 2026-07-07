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
	SaveOutputRecording  bool
	OutputRecordingFile  string
	MuteAudioPlayback    bool
	DisconnectGraceAfter time.Duration
}

func DefaultTeeConfig() TeeConfig {
	return TeeConfig{
		ChunkSize:            4096,
		Rate:                 16000,
		Channels:             1,
		RecordSeconds:        60000,
		OutputRecordingFile:  "./output_recording.wav",
		DisconnectGraceAfter: 5 * time.Second,
	}
}

type TranscriptionTeeClient struct {
	Clients []*Client
	Config  TeeConfig
}

func NewTranscriptionTeeClient(clients []*Client, cfg TeeConfig) (*TranscriptionTeeClient, error) {
	if len(clients) == 0 {
		return nil, fmt.Errorf("at least one client is required")
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
	if cfg.OutputRecordingFile != "" {
		base.OutputRecordingFile = cfg.OutputRecordingFile
	}
	base.SaveOutputRecording = cfg.SaveOutputRecording
	base.MuteAudioPlayback = cfg.MuteAudioPlayback
	if cfg.DisconnectGraceAfter != 0 {
		base.DisconnectGraceAfter = cfg.DisconnectGraceAfter
	}

	return &TranscriptionTeeClient{Clients: clients, Config: base}, nil
}

func (t *TranscriptionTeeClient) WaitForServerReady(ctx context.Context) error {
	for _, client := range t.Clients {
		for {
			if client.Recording() {
				break
			}
			if client.Waiting() || client.ServerError() {
				_ = t.CloseAllClients()
				return fmt.Errorf("server did not become ready")
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(25 * time.Millisecond):
			}
		}
	}
	return nil
}

func (t *TranscriptionTeeClient) CloseAllClients() error {
	var firstErr error
	for _, client := range t.Clients {
		if err := client.CloseWebSocket(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (t *TranscriptionTeeClient) WriteAllClientsSRT() error {
	var firstErr error
	for _, client := range t.Clients {
		if err := client.WriteSRTFile(""); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (t *TranscriptionTeeClient) MulticastPacket(packet []byte, unconditional bool) error {
	var firstErr error
	for _, client := range t.Clients {
		if unconditional || client.Recording() {
			if err := client.SendPacketToServer(packet); err != nil && firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

func (t *TranscriptionTeeClient) StreamPCM16LEReader(ctx context.Context, r io.Reader) error {
	reader := bufio.NewReader(r)
	chunk := make([]byte, t.Config.ChunkSize)

	for {
		if !anyClientRecording(t.Clients) {
			break
		}

		n, err := io.ReadFull(reader, chunk)
		if err != nil {
			if errorsIsEOFLike(err) {
				break
			}
			return err
		}

		audioFloat := bytesToFloat32PCM(chunk[:n])
		if err := t.MulticastPacket(float32ToBytes(audioFloat), false); err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	// Signal end-of-audio first so the backend can flush final segments before disconnect.
	_ = t.MulticastPacket([]byte(endOfAudioMarker), true)
	for _, client := range t.Clients {
		_ = client.WaitBeforeDisconnect(ctx)
	}
	_ = t.WriteAllClientsSRT()
	return t.CloseAllClients()
}

func (t *TranscriptionTeeClient) ProcessRTSPStream(ctx context.Context, rtspURL string) error {
	return t.processFFmpegStream(ctx, rtspURL, "", "")
}

func (t *TranscriptionTeeClient) ProcessHLSStream(ctx context.Context, hlsURL, saveFile string) error {
	return t.processFFmpegStream(ctx, hlsURL, "", saveFile)
}

func (t *TranscriptionTeeClient) processFFmpegStream(ctx context.Context, inputURL, inputFormat, saveFile string) error {
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

	if saveFile != "" {
		go saveInputStreamCopy(ctx, inputURL, saveFile)
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

func saveInputStreamCopy(ctx context.Context, inputURL, outputPath string) {
	cmd := exec.CommandContext(ctx, "ffmpeg", "-hide_banner", "-loglevel", "error", "-i", inputURL, "-c", "copy", "-f", "mpegts", outputPath)
	_ = cmd.Run()
}

func anyClientRecording(clients []*Client) bool {
	for _, c := range clients {
		if c.Recording() {
			return true
		}
	}
	return false
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
