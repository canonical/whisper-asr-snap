package messages

type SessionUpdateFormat struct {
	Rate int `json:"rate"`
}

type SessionUpdateTranscription struct {
	Model    string `json:"model,omitempty"`
	Language string `json:"language,omitempty"`
}

type SessionUpdateAudioInput struct {
	Format        SessionUpdateFormat        `json:"format"`
	Transcription SessionUpdateTranscription `json:"transcription"`
}

type SessionUpdateAudio struct {
	Input SessionUpdateAudioInput `json:"input"`
}

type SessionUpdateSession struct {
	Type         string             `json:"type"`
	Instructions string             `json:"instructions,omitempty"`
	Prompt       *string            `json:"prompt,omitempty"`
	Audio        SessionUpdateAudio `json:"audio"`
	Include      []string           `json:"include,omitempty"`
}

type SessionUpdate struct {
	MessageBase
	Session SessionUpdateSession `json:"session"`
}

func (m *SessionUpdate) New() {
	m.Type = "session.update"
}

func (m *SessionUpdate) Run() error {
	return nil
}
