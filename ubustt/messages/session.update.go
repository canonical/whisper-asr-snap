package messages

import (
	"fmt"
	"slices"
)

type SessionUpdateFormat struct {
	Rate int `json:"rate"`
}

type SessionUpdateTranscription struct {
	Model    *string `json:"model,omitempty"`
	Language *string `json:"language,omitempty"`
}

type SessionUpdateAudioInput struct {
	Format        *SessionUpdateFormat        `json:"format,omitempty"`
	Transcription *SessionUpdateTranscription `json:"transcription,omitempty"`
}

type SessionUpdateAudio struct {
	Input *SessionUpdateAudioInput `json:"input,omitempty"`
}

type SessionUpdateSession struct {
	Type         *string             `json:"type"`
	Instructions *string             `json:"instructions,omitempty"`
	Prompt       *string             `json:"prompt,omitempty"`
	Audio        *SessionUpdateAudio `json:"audio,omitempty"`
	Include      []string            `json:"include,omitempty"`
}

type SessionUpdate struct {
	MessageBase
	Session SessionUpdateSession `json:"session"`
}

func (m *SessionUpdate) New() {
	m.Type = "session.update"
}

func (m *SessionUpdate) Validate() error {
	var allowedTypeValues = []string{"realtime"}
	var allowedIncludeValues = []string{"item.input_audio_transcription.logprobs"}

	if m.Session.Type != nil && !slices.Contains(allowedTypeValues, *m.Session.Type) {
		return fmt.Errorf("invalid value for field \"type\": %q. Allowed values are: %v", *m.Session.Type, allowedTypeValues)
	}

	for _, includeItem := range m.Session.Include {
		if !slices.Contains(allowedIncludeValues, includeItem) {
			return fmt.Errorf("invalid value in \"include\": %q. Allowed values are: %v", includeItem, allowedIncludeValues)
		}
	}

	if m.Session.Audio != nil {
		if m.Session.Audio.Input == nil {
			return fmt.Errorf("provided object \"audio\" has no value")
		}

		if m.Session.Audio.Input.Format != nil {
			if m.Session.Audio.Input.Format.Rate <= 0 {
				return fmt.Errorf("expected a positive integer in \"audio.input.format.rate\", got %d", m.Session.Audio.Input.Format.Rate)
			}
		}

		if m.Session.Audio.Input.Transcription != nil {
			if m.Session.Audio.Input.Transcription.Model == nil && m.Session.Audio.Input.Transcription.Language == nil {
				return fmt.Errorf("provided object \"audio.input.transcription\" has no value")
			}
		}
	}

	return nil
}
