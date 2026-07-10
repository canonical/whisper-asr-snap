package messages

import "ubustt-proxy/backends"

type SessionUpdated struct {
	MessageBase
	Session SessionData `json:"session"`
}

func (m *SessionUpdated) New() {
	m.Type = "session.updated"
}

func NewSessionUpdated(cfg *backends.SessionConfig) *SessionUpdated {
	m := &SessionUpdated{
		Session: SessionData{
			Type:         &cfg.Type,
			Instructions: cfg.Instructions,
			Prompt:       cfg.Prompt,
			Include:      cfg.Include,
			Audio: &SessionAudio{
				Input: &SessionAudioInput{
					Format: &SessionAudioFormat{Rate: cfg.SampleRate},
					Transcription: &SessionTranscription{
						Model:    &cfg.Model,
						Language: &cfg.Lang,
					},
				},
			},
		},
	}
	m.New()
	return m
}
