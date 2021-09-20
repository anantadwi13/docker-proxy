package proxy

type Proxy interface {
	// Start will be run as blocking function
	Start() error

	// Shutdown proxy gracefully
	Shutdown() error
}
