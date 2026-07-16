package events

type ModelUnloaded struct {
	MessageBase
}

func (m *ModelUnloaded) New() {
	m.Type = "model.unloaded"
}
