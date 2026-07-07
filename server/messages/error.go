package messages

type ErrorDetail struct {
	Type    string `json:"type"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Error struct {
	MessageBase
	Error ErrorDetail `json:"error"`
}

func (m *Error) New() {
	m.Type = "error"
}

func (m *Error) Run() error {
	return nil
}
