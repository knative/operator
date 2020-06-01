package manifestival

import "github.com/go-logr/logr"

type Option func(*Manifest)

// UseLogger will cause manifestival to log its actions
func UseLogger(log logr.Logger) Option {
	return func(m *Manifest) {
		m.log = log
	}
}

// UseClient enables interaction with the k8s API server
func UseClient(client Client) Option {
	return func(m *Manifest) {
		m.Client = client
	}
}
