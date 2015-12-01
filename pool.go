package http

import (
	"crypto/tls"
	"net"
	"net/http"
	"sync"
	"time"
)

// ClientPool represents the behaviors that a HTTP Client Pool must satisfy.
type ClientPool interface {
	// SetTransport sets the transport to be shared by all the clients in the
	// pool. If nil, a default transport will be used. The default transport
	// will use the same settings as the default one in the core http package
	// plus the default TLS Configuration maintained in the pool.
	SetTransport(transport http.RoundTripper)

	// SetDefaultTLSConfig sets the TLS Configuration that will be used
	// by the default transport. A default transport will be used if no
	// transport has been specified.
	SetDefaultTLSConfig(tlsConfig *tls.Config)

	// GetClient returns a HTTP Client based on the specified timeout.
	GetClient(timeout time.Duration) *http.Client
}

// pool manages a set of HTTP clients for processing. A new Client is
// created for every different timeout options that is specified.
// Clients and Transports are safe for concurrent use by multiple
// goroutines and for efficiency should only be created once and re-used.
type pool struct {
	mtx       sync.RWMutex
	transport http.RoundTripper
	tlsConfig *tls.Config
	clients   map[time.Duration]*http.Client
}

// SetTransport sets the transport to be shared by all the clients in the
// pool. If nil, a default transport will be used. The default transport
// will use the same settings as the default one in the core http package
// plus the default TLS Configuration maintained in the pool.
func (p *pool) SetTransport(transport http.RoundTripper) {
	p.mtx.Lock()
	{
		p.transport = transport

		// Ensuring that new clients requested from the pool will use
		// the new transport settings.
		p.clients = make(map[time.Duration]*http.Client)
	}
	p.mtx.Unlock()
}

// SetDefaultTLSConfig sets the TLS Configuration that will be used
// by the default transport. A default transport will be used if no
// transport has been specified.
func (p *pool) SetDefaultTLSConfig(tlsConfig *tls.Config) {
	p.mtx.Lock()
	{
		p.tlsConfig = tlsConfig

		// Ensuring that new clients requested from the pool will use
		// the new transport settings.
		p.clients = make(map[time.Duration]*http.Client)
	}
	p.mtx.Unlock()
}

// GetClient returns a HTTP Client for making HTTP calls based
// on the specified timeout.
func (p *pool) GetClient(timeout time.Duration) *http.Client {
	// Locate a client for this timeout.
	p.mtx.RLock()
	{
		if client := p.clients[timeout]; client != nil {
			p.mtx.RUnlock()
			return client
		}
	}
	p.mtx.RUnlock()

	// Create a new client for this timeout if one did not exist.
	var client *http.Client

	p.mtx.Lock()
	{
		// Check again to be safe now that we are in the write lock.
		if client = p.clients[timeout]; client == nil {
			transport := p.transport
			if transport == nil {
				// Create our own transport using the same settings as
				// the default one in the core http package plus the
				// default TLS Configuration maintained in the pool.
				// This maintains a pool of connections.
				transport = &http.Transport{
					Proxy:           http.ProxyFromEnvironment,
					TLSClientConfig: p.tlsConfig,
					Dial: (&net.Dialer{
						Timeout:   30 * time.Second,
						KeepAlive: 30 * time.Second,
					}).Dial,
					TLSHandshakeTimeout: 10 * time.Second,
				}
			}

			// Create a new Client to use this transport
			// for this specific timeout.
			client = &http.Client{
				Transport: transport,
				Timeout:   timeout,
			}

			// Save this client to the map.
			p.clients[timeout] = client
		}
	}
	p.mtx.Unlock()

	return client
}

// NewClientPool returns a new, empty ClientPool.
func NewClientPool() ClientPool {
	return &pool{
		clients: make(map[time.Duration]*http.Client),
	}
}

// DefaultClientPool represents the default pool for managing HTTP Clients.
var DefaultClientPool = NewClientPool()
