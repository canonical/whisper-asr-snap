package whisperlive

type TranscriptionClientConfig struct {
	ClientConfig
}

type TranscriptionClient struct {
	Client *Client
	Tee    *TranscriptionTeeClient
}

func NewTranscriptionClient(cfg TranscriptionClientConfig) (*TranscriptionClient, error) {
	ccfg := cfg.ClientConfig

	client, err := NewClient(ccfg)
	if err != nil {
		return nil, err
	}

	teeCfg := DefaultTeeConfig()

	tee, err := NewTranscriptionTeeClient(client, teeCfg)
	if err != nil {
		_ = client.CloseWebSocket()
		return nil, err
	}

	return &TranscriptionClient{Client: client, Tee: tee}, nil
}
