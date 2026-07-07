package messages

import (
	"encoding/json"
	"fmt"
)

type MessageBase struct {
	message
	Type string `json:"type"`
}

type message interface {
	New()
	Run() error
}

func FromJson(jsonData []byte) (MessageBase, error) {
	var msg MessageBase
	err := json.Unmarshal(jsonData, &msg)
	if err != nil {
		return MessageBase{}, fmt.Errorf("unmarshaling JSON: %w", err)
	}

	var dst message
	switch msg.Type {

	case "session.created":
		dst = new(SessionCreated)
	case "session.update":
		dst = new(SessionUpdate)
	case "session.updated":
		dst = new(SessionUpdated)
	case "input_audio_buffer.append":
		dst = new(InputAudioBufferAppend)
	case "input_audio_buffer.commit":
		dst = new(InputAudioBufferCommit)
	case "conversation.item.input_audio_transcription.delta":
		dst = new(ConversationItemInputAudioTranscriptionDelta)
	case "conversation.item.input_audio_transcription.completed":
		dst = new(ConversationItemInputAudioTranscriptionCompleted)
	case "error":
		dst = new(Error)
	default:
		return MessageBase{}, fmt.Errorf("unknown message type: %s", msg.Type)
	}

	if err := json.Unmarshal(jsonData, dst); err != nil {
		return MessageBase{}, fmt.Errorf("unmarshaling JSON: %w", err)
	}
	return MessageBase{message: dst, Type: msg.Type}, nil
}
