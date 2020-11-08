// Package pool implements a pool of net.Conn interfaces to manage and reuse them.
package pool

import (
	"errors"
	"fmt"
	"net"
	"sync"
)

// Factory is a function to create new connections.
type Factory func() (net.Conn, error)

// Factory is a function to create new connections.
type Pool struct {
	// storage for our net.Conn connections
	conns chan net.Conn

	// net.Conn generator
	factory Factory

	mu sync.Mutex // protects isDesroyed field
}

// New returns a new pool with an initial capacity and maximum capacity.
// Factory is used when initial capacity is greater then zero to fill the  pool.
func New(initalCap, maxCap int, factory Factory) (*Pool, error) {
	if initalCap <= 0 || maxCap <= 0 || initalCap > maxCap {
		return nil, errors.New("invalid capacity settings")
	}

	p := &Pool{
		conns: make(chan net.Conn, maxCap),
		factory: factory,
	}

	// create initial connections, if something goes wrong,
	// just close the pool error out.
	for i := 0; i < initalCap; i++ {
		conn, err := factory()
		if err != nil {
			p.Close()
			return nil, fmt.Errorf("factory is not able to fill the pool: %s", err)
		}
		p.conns <- conn
	}

	return p, nil
}

func (p *Pool) getConns() chan net.Conn {
	p.mu.Lock()
	conns := p.conns
	p.mu.Unlock()
	return conns
}

// Get returns a new connection from the pool. After using the connection it
// should be put back via the Put() method. If there is no new connection
// available in the pool, a new connection will be created via the Factory()
// method.
func (p *Pool) Get() (net.Conn, error) {
	conns := p.getConns()
	if conns == nil {
		return nil, errors.New("pool is closed")
	}

	select {
	case conn := <- p.conns:
		if conn == nil {
			return nil, errors.New("pool is closed")
		}
		return conn, nil
	default:
		return p.factory()
	}
}

// Put puts an existing connection into the pool. If the pool is full or closed, conn is
// simply closed.
func (p *Pool) Put(conn net.Conn) {
	if conn == nil {
		return
	}
	if !p.put(conn) {
		_ = conn.Close()
	}
}

func (p *Pool) put(conn net.Conn) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.conns == nil {
		return false
	}

	select {
	case p.conns <- conn:
		return true
	default:
	}
	return false
}

// Close closes the pool and all its connections. After Close() the
// pool is no longer usable.
func (p *Pool) Close() {
	conns := p.closePool()
	if conns == nil {
		return
	}
	close(conns)
	for conn := range conns {
		_ = conn.Close()
	}
}

func (p *Pool) closePool() chan net.Conn {
	p.mu.Lock()
	defer p.mu.Unlock()
	conns := p.conns
	p.conns = nil
	p.factory = nil
	return conns
}

// MaximumCapacity returns the maximum capacity of the pool
func (p *Pool) MaximumCapacity() int {
	return cap(p.conns)
}
// UsedCapacity returns the used capacity of the pool.
func (p *Pool) UsedCapacity() int {
	return len(p.conns)
}