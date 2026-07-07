package whisperlive

import (
	"fmt"
	"strings"
)

type TranscriptionClientConfig struct {
	ClientConfig
	SaveOutputRecording    bool
	OutputRecordingFile    string
	OutputTranscriptionSRT string
	MuteAudioPlayback      bool
}

type TranscriptionClient struct {
	Client *Client
	Tee    *TranscriptionTeeClient
}

func NewTranscriptionClient(cfg TranscriptionClientConfig) (*TranscriptionClient, error) {
	if cfg.SaveOutputRecording && !strings.HasSuffix(cfg.OutputRecordingFile, ".wav") {
		return nil, fmt.Errorf("please provide a valid output recording filename ending with .wav")
	}
	if cfg.OutputTranscriptionSRT != "" && !strings.HasSuffix(cfg.OutputTranscriptionSRT, ".srt") {
		return nil, fmt.Errorf("please provide a valid output transcription path ending with .srt")
	}
	if cfg.TranslationSRTFilePath != nil && *cfg.TranslationSRTFilePath != "" && !strings.HasSuffix(*cfg.TranslationSRTFilePath, ".srt") {
		return nil, fmt.Errorf("please provide a valid translation srt file path ending with .srt")
	}

	ccfg := cfg.ClientConfig
	if cfg.OutputTranscriptionSRT != "" {
		ccfg.SRTFilePath = new(cfg.OutputTranscriptionSRT)
	}

	client, err := NewClient(ccfg)
	if err != nil {
		return nil, err
	}

	teeCfg := DefaultTeeConfig()
	teeCfg.SaveOutputRecording = cfg.SaveOutputRecording
	if cfg.OutputRecordingFile != "" {
		teeCfg.OutputRecordingFile = cfg.OutputRecordingFile
	}
	teeCfg.MuteAudioPlayback = cfg.MuteAudioPlayback

	tee, err := NewTranscriptionTeeClient([]*Client{client}, teeCfg)
	if err != nil {
		_ = client.CloseWebSocket()
		return nil, err
	}

	return &TranscriptionClient{Client: client, Tee: tee}, nil
}
