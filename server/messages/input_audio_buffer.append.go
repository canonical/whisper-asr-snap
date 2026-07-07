package messages

type InputAudioBufferAppend struct {
	MessageBase
	Audio string `json:"audio"`
}

func (m *InputAudioBufferAppend) New() {
	m.Type = "input_audio_buffer.append"
}

func (m *InputAudioBufferAppend) Run() error {
	return nil
}
