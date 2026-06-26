package config

type APIConfig struct {
	// PprofSecret is an optional shared secret for authenticating pprof
	// endpoints as an alternative to JWT-based admin auth. When set,
	// requests must provide the secret as a Bearer token in the
	// Authorization header.
	PprofSecret string `config:"pprof_secret"`
}

func (A APIConfig) Defaults() map[string]any {
	return map[string]any{
		"PprofSecret": "",
	}
}
