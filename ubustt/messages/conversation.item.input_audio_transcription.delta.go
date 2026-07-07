package messages

type DeltaLogprob struct {
	Token   string  `json:"token"`
	Logprob float64 `json:"logprob"`
	Bytes   []int   `json:"bytes,omitempty"`
}

type ConversationItemInputAudioTranscriptionDelta struct {
	MessageBase
	ItemID       string         `json:"item_id"`
	ContentIndex int            `json:"content_index"`
	Delta        string         `json:"delta"`
	Logprobs     []DeltaLogprob `json:"logprobs,omitempty"`
}

func (m *ConversationItemInputAudioTranscriptionDelta) New() {
	m.Type = "conversation.item.input_audio_transcription.delta"
}
