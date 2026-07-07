package messages

type InputAudioBufferCommit struct {
	MessageBase
}

func (m *InputAudioBufferCommit) New() {
	m.Type = "input_audio_buffer.commit"
}
