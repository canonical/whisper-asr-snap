package events

type ModelLoaded struct {
	MessageBase
}

func (m *ModelLoaded) New() {
	m.Type = "model.loaded"
}
