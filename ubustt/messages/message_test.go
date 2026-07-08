package messages

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestFromJsonSessionCreated(t *testing.T) {
	prompt := "prompt"
	want := &SessionCreated{
		Session: SessionCreatedSession{
			Type:         "realtime",
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
	got, ok := msg.(*SessionCreated)
	if !ok {
		t.Fatalf("message is %T, want *SessionCreated", msg)
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

func TestFromJsonUnknownType(t *testing.T) {
	data := []byte(`{"type":"unknown.event"}`)
	_, err := FromJson(data)
	if err == nil {
		t.Fatal("expected error for unknown type, got nil")
	}
	if !strings.Contains(err.Error(), "unknown message type") {
		t.Errorf("error = %q, want to contain 'unknown message type'", err.Error())
	}
}

func TestFromJsonMalformed(t *testing.T) {
	_, err := FromJson([]byte(`{not valid json`))
	if err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}
}

func TestFromJsonSessionUpdate(t *testing.T) {
	prompt := "my-prompt"
	want := &SessionUpdate{
		Session: SessionUpdateSession{
			Type:         "realtime",
			Instructions: "do things",
			Prompt:       &prompt,
			Audio: SessionUpdateAudio{
				Input: SessionUpdateAudioInput{
					Format: SessionUpdateFormat{Rate: 16000},
					Transcription: SessionUpdateTranscription{
						Model:    "large",
						Language: "fr",
					},
				},
			},
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
	got, ok := msg.(*SessionUpdate)
	if !ok {
		t.Fatalf("got %T, want *SessionUpdate", msg)
	}
	if got.Type != "session.update" {
		t.Errorf("Type = %q, want %q", got.Type, "session.update")
	}
	if got.Session.Audio.Input.Transcription.Language != "fr" {
		t.Errorf("Language = %q, want %q", got.Session.Audio.Input.Transcription.Language, "fr")
	}
	if got.Session.Prompt == nil || *got.Session.Prompt != prompt {
		t.Errorf("Prompt = %v, want %q", got.Session.Prompt, prompt)
	}
}

func TestFromJsonSessionUpdated(t *testing.T) {
	want := &SessionUpdated{
		Session: SessionUpdatedSession{
			Type: "realtime",
			Audio: SessionUpdatedAudio{
				Input: SessionUpdatedAudioInput{
					Format: SessionUpdatedFormat{Rate: 44100},
					Transcription: SessionUpdatedTranscription{
						Model:    "small",
						Language: "de",
					},
				},
			},
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
	got, ok := msg.(*SessionUpdated)
	if !ok {
		t.Fatalf("got %T, want *SessionUpdated", msg)
	}
	if got.Type != "session.updated" {
		t.Errorf("Type = %q, want %q", got.Type, "session.updated")
	}
	if got.Session.Audio.Input.Format.Rate != 44100 {
		t.Errorf("Rate = %d, want 44100", got.Session.Audio.Input.Format.Rate)
	}
}

func TestFromJsonInputAudioBufferAppend(t *testing.T) {
	want := &InputAudioBufferAppend{Audio: "base64audiodata=="}
	want.New()

	data, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("marshaling: %v", err)
	}
	msg, err := FromJson(data)
	if err != nil {
		t.Fatalf("FromJson: %v", err)
	}
	got, ok := msg.(*InputAudioBufferAppend)
	if !ok {
		t.Fatalf("got %T, want *InputAudioBufferAppend", msg)
	}
	if got.Type != "input_audio_buffer.append" {
		t.Errorf("Type = %q, want %q", got.Type, "input_audio_buffer.append")
	}
	if got.Audio != "base64audiodata==" {
		t.Errorf("Audio = %q, want %q", got.Audio, "base64audiodata==")
	}
}

func TestFromJsonInputAudioBufferCommit(t *testing.T) {
	want := &InputAudioBufferCommit{}
	want.New()

	data, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("marshaling: %v", err)
	}
	msg, err := FromJson(data)
	if err != nil {
		t.Fatalf("FromJson: %v", err)
	}
	got, ok := msg.(*InputAudioBufferCommit)
	if !ok {
		t.Fatalf("got %T, want *InputAudioBufferCommit", msg)
	}
	if got.Type != "input_audio_buffer.commit" {
		t.Errorf("Type = %q, want %q", got.Type, "input_audio_buffer.commit")
	}
}

func TestFromJsonTranscriptionDelta(t *testing.T) {
	want := &ConversationItemInputAudioTranscriptionDelta{
		Delta: "hello",
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
	got, ok := msg.(*ConversationItemInputAudioTranscriptionDelta)
	if !ok {
		t.Fatalf("got %T, want *ConversationItemInputAudioTranscriptionDelta", msg)
	}
	if got.Type != "conversation.item.input_audio_transcription.delta" {
		t.Errorf("Type = %q", got.Type)
	}
	if got.Delta != "hello" {
		t.Errorf("Delta = %q, want %q", got.Delta, "hello")
	}
}

func TestFromJsonTranscriptionCompleted(t *testing.T) {
	want := &ConversationItemInputAudioTranscriptionCompleted{
		Transcript: "the full transcript",
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
	got, ok := msg.(*ConversationItemInputAudioTranscriptionCompleted)
	if !ok {
		t.Fatalf("got %T, want *ConversationItemInputAudioTranscriptionCompleted", msg)
	}
	if got.Type != "conversation.item.input_audio_transcription.completed" {
		t.Errorf("Type = %q", got.Type)
	}
	if got.Transcript != "the full transcript" {
		t.Errorf("Transcript = %q, want %q", got.Transcript, "the full transcript")
	}
}

func TestFromJsonError(t *testing.T) {
	want := &Error{
		Error: ErrorDetail{
			Type:    ErrorTypeInvalidRequest,
			Code:    ErrorCodeInvalidParameter,
			Message: "bad param",
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
	got, ok := msg.(*Error)
	if !ok {
		t.Fatalf("got %T, want *Error", msg)
	}
	if got.Type != "error" {
		t.Errorf("Type = %q, want %q", got.Type, "error")
	}
	if got.Error.Code != ErrorCodeInvalidParameter {
		t.Errorf("Code = %q, want %q", got.Error.Code, ErrorCodeInvalidParameter)
	}
	if got.Error.Message != "bad param" {
		t.Errorf("Message = %q, want %q", got.Error.Message, "bad param")
	}
}

func TestToJson(t *testing.T) {
	m := &InputAudioBufferCommit{}
	// Do NOT call New() — ToJson must set the type.
	data, err := ToJson(m)
	if err != nil {
		t.Fatalf("ToJson: %v", err)
	}
	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if obj["type"] != "input_audio_buffer.commit" {
		t.Errorf("type = %v, want %q", obj["type"], "input_audio_buffer.commit")
	}
}

func TestNewSessionCreated(t *testing.T) {
	m := NewSessionCreated("large", "es", 16000)
	if m.Type != "session.created" {
		t.Errorf("Type = %q, want %q", m.Type, "session.created")
	}
	if m.Session.Audio.Input.Transcription.Model != "large" {
		t.Errorf("Model = %q, want %q", m.Session.Audio.Input.Transcription.Model, "large")
	}
	if m.Session.Audio.Input.Transcription.Language != "es" {
		t.Errorf("Language = %q, want %q", m.Session.Audio.Input.Transcription.Language, "es")
	}
	if m.Session.Audio.Input.Format.Rate != 16000 {
		t.Errorf("Rate = %d, want 16000", m.Session.Audio.Input.Format.Rate)
	}
}

func TestNewSessionUpdated(t *testing.T) {
	prompt := "p"
	src := SessionUpdateSession{
		Type:         "realtime",
		Instructions: "inst",
		Prompt:       &prompt,
		Audio: SessionUpdateAudio{
			Input: SessionUpdateAudioInput{
				Format: SessionUpdateFormat{Rate: 8000},
				Transcription: SessionUpdateTranscription{
					Model:    "tiny",
					Language: "ja",
				},
			},
		},
		Include: []string{"logprobs"},
	}
	m := NewSessionUpdated(src)
	if m.Type != "session.updated" {
		t.Errorf("Type = %q, want %q", m.Type, "session.updated")
	}
	if m.Session.Instructions != "inst" {
		t.Errorf("Instructions = %q, want %q", m.Session.Instructions, "inst")
	}
	if m.Session.Prompt == nil || *m.Session.Prompt != "p" {
		t.Errorf("Prompt = %v, want %q", m.Session.Prompt, "p")
	}
	if m.Session.Audio.Input.Format.Rate != 8000 {
		t.Errorf("Rate = %d, want 8000", m.Session.Audio.Input.Format.Rate)
	}
	if m.Session.Audio.Input.Transcription.Model != "tiny" {
		t.Errorf("Model = %q, want %q", m.Session.Audio.Input.Transcription.Model, "tiny")
	}
	if len(m.Session.Include) != 1 || m.Session.Include[0] != "logprobs" {
		t.Errorf("Include = %v, want [logprobs]", m.Session.Include)
	}
}

func TestNewTranscriptionDelta(t *testing.T) {
	m := NewTranscriptionDelta("partial text")
	if m.Type != "conversation.item.input_audio_transcription.delta" {
		t.Errorf("Type = %q", m.Type)
	}
	if m.Delta != "partial text" {
		t.Errorf("Delta = %q", m.Delta)
	}
}

func TestNewTranscriptionCompleted(t *testing.T) {
	m := NewTranscriptionCompleted("final text")
	if m.Type != "conversation.item.input_audio_transcription.completed" {
		t.Errorf("Type = %q", m.Type)
	}
	if m.Transcript != "final text" {
		t.Errorf("Transcript = %q", m.Transcript)
	}
}

func TestNewError(t *testing.T) {
	m := NewError(ErrorTypeServer, ErrorCodeServerError, "something broke")
	if m.Type != "error" {
		t.Errorf("Type = %q, want %q", m.Type, "error")
	}
	if m.Error.Type != ErrorTypeServer {
		t.Errorf("Error.Type = %q, want %q", m.Error.Type, ErrorTypeServer)
	}
	if m.Error.Code != ErrorCodeServerError {
		t.Errorf("Error.Code = %q, want %q", m.Error.Code, ErrorCodeServerError)
	}
	if m.Error.Message != "something broke" {
		t.Errorf("Error.Message = %q, want %q", m.Error.Message, "something broke")
	}
}

func TestFromJsonModelLoaded(t *testing.T) {
	want := &ModelLoaded{}
	want.New()

	data, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("marshaling: %v", err)
	}
	msg, err := FromJson(data)
	if err != nil {
		t.Fatalf("FromJson: %v", err)
	}
	got, ok := msg.(*ModelLoaded)
	if !ok {
		t.Fatalf("got %T, want *ModelLoaded", msg)
	}
	if got.Type != "model.loaded" {
		t.Errorf("Type = %q, want %q", got.Type, "model.loaded")
	}
}

func TestFromJsonModelUnloaded(t *testing.T) {
	want := &ModelUnloaded{}
	want.New()

	data, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("marshaling: %v", err)
	}
	msg, err := FromJson(data)
	if err != nil {
		t.Fatalf("FromJson: %v", err)
	}
	got, ok := msg.(*ModelUnloaded)
	if !ok {
		t.Fatalf("got %T, want *ModelUnloaded", msg)
	}
	if got.Type != "model.unloaded" {
		t.Errorf("Type = %q, want %q", got.Type, "model.unloaded")
	}
}
