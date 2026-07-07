package messages

type ModelLoaded struct {
	MessageBase
}

func (m *ModelLoaded) New() {
	m.Type = "model.loaded"
}
