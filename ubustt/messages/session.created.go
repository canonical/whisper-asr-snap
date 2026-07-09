package messages

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
	Instructions string              `json:"instructions,omitempty"`
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

// NewSessionCreated builds the default session advertised to a client right
// after it connects. Values typically come from the active backend config.
func NewSessionCreated(model, language string, rate int) *SessionCreated {
	m := &SessionCreated{
		Session: SessionCreatedSession{
			Type: "realtime",
			Audio: SessionCreatedAudio{
				Input: SessionCreatedAudioInput{
					Format: SessionCreatedFormat{Rate: rate},
					Transcription: SessionCreatedTranscription{
						Model:    model,
						Language: language,
					},
				},
			},
		},
	}
	m.New()
	return m
}
