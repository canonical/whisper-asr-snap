package whisperlive

import "time"

type ClientConfig struct {
	Host                  *string
	Port                  *int
	Lang                  *string
	Translate             *bool
	Model                 *string
	UseVAD                *bool
	UseWSS                *bool
	LogTranscription      *bool
	SendLastNSegments     *int
	NoSpeechThresh        *float64
	ClipAudio             *bool
	SameOutputThreshold   *int
	TranscriptionCallback TranscriptionCallback
	EnableTranslation     *bool
	TargetLanguage        *string
	TranslationCallback   TranslationCallback
	EnableTimestamps      *bool
	DisplaySegments       *int
	Hotwords              *string
	EnableDiarization     *bool
	MaxSpeakers           *int
	WordTimestamps        *bool
	MaxRetries            *int
	RetryDelay            *time.Duration
	InitialPrompt         *string
	VADParameters         map[string]any
	DisconnectAfterIdle   *time.Duration
	UID                   *string
	AudioFormat           *string
}

func DefaultClientConfig() ClientConfig {
	return ClientConfig{
		Model:               new("small"),
		UseVAD:              new(true),
		LogTranscription:    new(true),
		SendLastNSegments:   new(10),
		NoSpeechThresh:      new(0.45),
		SameOutputThreshold: new(10),
		TargetLanguage:      new("fr"),
		DisplaySegments:     new(4),
		MaxSpeakers:         new(10),
		RetryDelay:          new(5 * time.Second),
		DisconnectAfterIdle: new(15 * time.Second),
		AudioFormat:         new("float32"),
	}
}

func mergeConfig(base, override ClientConfig) ClientConfig {
	cfg := base
	if override.Host != nil {
		cfg.Host = override.Host
	}
	if override.Port != nil {
		cfg.Port = override.Port
	}
	if override.Lang != nil {
		cfg.Lang = override.Lang
	}
	if override.Translate != nil {
		cfg.Translate = override.Translate
	}
	if override.Model != nil {
		cfg.Model = override.Model
	}
	if override.UseVAD != nil {
		cfg.UseVAD = override.UseVAD
	}
	if override.UseWSS != nil {
		cfg.UseWSS = override.UseWSS
	}
	if override.LogTranscription != nil {
		cfg.LogTranscription = override.LogTranscription
	}
	if override.SendLastNSegments != nil {
		cfg.SendLastNSegments = override.SendLastNSegments
	}
	if override.NoSpeechThresh != nil {
		cfg.NoSpeechThresh = override.NoSpeechThresh
	}
	if override.ClipAudio != nil {
		cfg.ClipAudio = override.ClipAudio
	}
	if override.SameOutputThreshold != nil {
		cfg.SameOutputThreshold = override.SameOutputThreshold
	}
	if override.TranscriptionCallback != nil {
		cfg.TranscriptionCallback = override.TranscriptionCallback
	}
	if override.EnableTranslation != nil {
		cfg.EnableTranslation = override.EnableTranslation
	}
	if override.TargetLanguage != nil {
		cfg.TargetLanguage = override.TargetLanguage
	}
	if override.TranslationCallback != nil {
		cfg.TranslationCallback = override.TranslationCallback
	}
	if override.EnableTimestamps != nil {
		cfg.EnableTimestamps = override.EnableTimestamps
	}
	if override.DisplaySegments != nil {
		cfg.DisplaySegments = override.DisplaySegments
	}
	if override.Hotwords != nil {
		cfg.Hotwords = override.Hotwords
	}
	if override.EnableDiarization != nil {
		cfg.EnableDiarization = override.EnableDiarization
	}
	if override.MaxSpeakers != nil {
		cfg.MaxSpeakers = override.MaxSpeakers
	}
	if override.WordTimestamps != nil {
		cfg.WordTimestamps = override.WordTimestamps
	}
	if override.MaxRetries != nil {
		cfg.MaxRetries = override.MaxRetries
	}
	if override.RetryDelay != nil {
		cfg.RetryDelay = override.RetryDelay
	}
	if override.InitialPrompt != nil {
		cfg.InitialPrompt = override.InitialPrompt
	}
	if override.VADParameters != nil {
		cfg.VADParameters = override.VADParameters
	}
	if override.DisconnectAfterIdle != nil {
		cfg.DisconnectAfterIdle = override.DisconnectAfterIdle
	}
	if override.UID != nil {
		cfg.UID = override.UID
	}
	if override.AudioFormat != nil {
		cfg.AudioFormat = override.AudioFormat
	}
	return cfg
}
