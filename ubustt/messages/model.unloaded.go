package messages

type ModelUnloaded struct {
	MessageBase
}

func (m *ModelUnloaded) New() {
	m.Type = "model.unloaded"
}
