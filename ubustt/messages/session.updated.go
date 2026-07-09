package messages

import "ubustt-proxy/backends"

type SessionUpdatedFormat struct {
	Rate int `json:"rate"`
}

type SessionUpdatedTranscription struct {
	Model    string `json:"model"`
	Language string `json:"language,omitempty"`
}

type SessionUpdatedAudioInput struct {
	Format        SessionUpdatedFormat        `json:"format"`
	Transcription SessionUpdatedTranscription `json:"transcription"`
}

type SessionUpdatedAudio struct {
	Input SessionUpdatedAudioInput `json:"input"`
}

type SessionUpdatedSession struct {
	Type         string              `json:"type"`
	Instructions *string             `json:"instructions,omitempty"`
	Prompt       *string             `json:"prompt,omitempty"`
	Audio        SessionUpdatedAudio `json:"audio"`
	Include      []string            `json:"include,omitempty"`
}

type SessionUpdated struct {
	MessageBase
	Session SessionUpdatedSession `json:"session"`
}

func (m *SessionUpdated) New() {
	m.Type = "session.updated"
}

func NewSessionUpdated(cfg *backends.SessionConfig) *SessionUpdated {
	m := &SessionUpdated{
		Session: SessionUpdatedSession{
			Type:         cfg.Type,
			Instructions: cfg.Instructions,
			Prompt:       cfg.Prompt,
			Include:      cfg.Include,
			Audio: SessionUpdatedAudio{
				Input: SessionUpdatedAudioInput{
					Format: SessionUpdatedFormat{Rate: cfg.SampleRate},
					Transcription: SessionUpdatedTranscription{
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
