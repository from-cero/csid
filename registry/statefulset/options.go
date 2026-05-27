package statefulset

import "os"

// Option is a functional option for Registry.
type Option func(*config)

type config struct {
	podNameFn func() (string, error)
}

func defaultConfig() config {
	return config{
		podNameFn: os.Hostname,
	}
}

// WithPodName sets a fixed pod name instead of reading os.Hostname.
// Useful for testing or when the pod name is injected via an environment variable
// rather than the hostname (e.g. via the Downward API).
func WithPodName(name string) Option {
	return func(c *config) {
		c.podNameFn = func() (string, error) { return name, nil }
	}
}

// WithPodNameFunc sets a custom function to resolve the pod name.
// The function is called once per Acquire.
func WithPodNameFunc(fn func() (string, error)) Option {
	return func(c *config) { c.podNameFn = fn }
}
