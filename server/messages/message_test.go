package messages

import (
	"encoding/json"
	"testing"
)

func TestFromJsonSessionCreated(t *testing.T) {
	prompt := "prompt"
	want := &SessionCreated{
		Session: SessionCreatedSession{
			Type:         "transcription",
			Instructions: "instructions",
			Prompt:       &prompt,
			Audio: SessionCreatedAudio{
				Input: SessionCreatedAudioInput{
					Format: SessionCreatedFormat{Rate: 16000},
					Transcription: SessionCreatedTranscription{
						Model:    "small",
						Language: "en",
					},
				},
			},
			Include: []string{"logprobs"},
		},
	}
	want.New()

	data, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("marshaling: %v", err)
	}

	msg, err := FromJson(data)
	if err != nil {
		t.Fatalf("FromJson: %v", err)
	}

	// Check data type
	got, ok := msg.message.(*SessionCreated)
	if !ok {
		t.Fatalf("message is %T, want *SessionCreated", msg.message)
	}

	// Check fields
	if got.Type != "session.created" {
		t.Errorf("got.Type = %q, want %q", got.Type, "session.created")
	}
	if got.Session.Type != want.Session.Type {
		t.Errorf("Session.Type = %q, want %q", got.Session.Type, want.Session.Type)
	}
	if got.Session.Instructions != want.Session.Instructions {
		t.Errorf("Session.Instructions = %q, want %q", got.Session.Instructions, want.Session.Instructions)
	}
	if got.Session.Prompt == nil || *got.Session.Prompt != prompt {
		t.Errorf("Session.Prompt = %v, want %q", got.Session.Prompt, prompt)
	}
	if got.Session.Audio.Input.Format.Rate != want.Session.Audio.Input.Format.Rate {
		t.Errorf("Format.Rate = %d, want %d", got.Session.Audio.Input.Format.Rate, want.Session.Audio.Input.Format.Rate)
	}
	if got.Session.Audio.Input.Transcription.Model != want.Session.Audio.Input.Transcription.Model {
		t.Errorf("Transcription.Model = %q, want %q", got.Session.Audio.Input.Transcription.Model, want.Session.Audio.Input.Transcription.Model)
	}
	if got.Session.Audio.Input.Transcription.Language != want.Session.Audio.Input.Transcription.Language {
		t.Errorf("Transcription.Language = %q, want %q", got.Session.Audio.Input.Transcription.Language, want.Session.Audio.Input.Transcription.Language)
	}
	if len(got.Session.Include) != 1 || got.Session.Include[0] != "logprobs" {
		t.Errorf("Session.Include = %v, want [logprobs]", got.Session.Include)
	}
}
