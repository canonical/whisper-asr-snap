package messages

import "ubustt-proxy/backends"

type SessionCreatedFormat struct {
	Rate int `json:"rate"`
}

type SessionCreatedTranscription struct {
	Model    string `json:"model"`
	Language string `json:"language,omitempty"`
}

type SessionCreatedAudioInput struct {
	Format        SessionCreatedFormat        `json:"format"`
	Transcription SessionCreatedTranscription `json:"transcription"`
}

type SessionCreatedAudio struct {
	Input SessionCreatedAudioInput `json:"input"`
}

type SessionCreatedSession struct {
	Type         string              `json:"type"`
	Instructions *string             `json:"instructions,omitempty"`
	Prompt       *string             `json:"prompt,omitempty"`
	Audio        SessionCreatedAudio `json:"audio"`
	Include      []string            `json:"include,omitempty"`
}

type SessionCreated struct {
	MessageBase
	Session SessionCreatedSession `json:"session"`
}

func (m *SessionCreated) New() {
	m.Type = "session.created"
}

func NewSessionCreated(cfg *backends.SessionConfig) *SessionCreated {
	m := &SessionCreated{
		Session: SessionCreatedSession{
			Type:         cfg.Type,
			Instructions: cfg.Instructions,
			Prompt:       cfg.Prompt,
			Include:      cfg.Include,
			Audio: SessionCreatedAudio{
				Input: SessionCreatedAudioInput{
					Format: SessionCreatedFormat{Rate: cfg.SampleRate},
					Transcription: SessionCreatedTranscription{
						Model:    cfg.Model,
						Language: cfg.Lang,
					},
				},
			},
		},
	}
	m.New()
	return m
}
