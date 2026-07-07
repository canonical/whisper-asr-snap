package messages

type CompletedLogprob struct {
	Token   string  `json:"token"`
	Logprob float64 `json:"logprob"`
	Bytes   []int   `json:"bytes,omitempty"`
}

type ConversationItemInputAudioTranscriptionCompleted struct {
	MessageBase
	ItemID       string             `json:"item_id"`
	ContentIndex int                `json:"content_index"`
	Transcript   string             `json:"transcript"`
	Logprobs     []CompletedLogprob `json:"logprobs,omitempty"`
}

func (m *ConversationItemInputAudioTranscriptionCompleted) New() {
	m.Type = "conversation.item.input_audio_transcription.completed"
}

// NewTranscriptionCompleted builds the terminal event carrying the finalized
// transcript for a committed audio segment.
func NewTranscriptionCompleted(itemID string, contentIndex int, transcript string) *ConversationItemInputAudioTranscriptionCompleted {
	m := &ConversationItemInputAudioTranscriptionCompleted{
		ItemID:       itemID,
		ContentIndex: contentIndex,
		Transcript:   transcript,
	}
	m.New()
	return m
}
