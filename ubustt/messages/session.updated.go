package messages

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
	Instructions string              `json:"instructions,omitempty"`
	Prompt       *string             `json:"prompt"`
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
