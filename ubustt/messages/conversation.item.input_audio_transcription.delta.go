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

// NewTranscriptionDelta builds an incremental transcript fragment event.
func NewTranscriptionDelta(itemID string, contentIndex int, delta string) *ConversationItemInputAudioTranscriptionDelta {
	m := &ConversationItemInputAudioTranscriptionDelta{
		ItemID:       itemID,
		ContentIndex: contentIndex,
		Delta:        delta,
	}
	m.New()
	return m
}
