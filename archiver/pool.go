package archiver

import (
	"fmt"
	"net"
	"sync/atomic"
)

// For handling connections to the TSDB, we want to have a pool of long-lived
// connection objects that escape Go's garbage collection cycles. Objects
// stored in sync.Pool that are not referenced are garbage collected by Go, so
// we want to use a buffered channel to maintain non GC-able references to
// connections. When a coalesced buffer wants to write to a TSDB, it can grab a
// connection from the channel, and when it is finished, it can return it to
// the channel. Buffered channels give us a way to place a maximum number of
// connections as well, which is nice.

type tsConn struct {
	conn   net.Conn
	closed bool
}

func (c *tsConn) Read(b []byte) (int, error) {
	return c.conn.Read(b)
}

func (c *tsConn) Write(b []byte) (int, error) {
	return c.conn.Write(b)
}

func (c *tsConn) Close() error {
	c.closed = true
	return c.conn.Close()
}

func (c *tsConn) IsClosed() bool {
	return c.closed
}

type connectionPool struct {
	pool chan *tsConn
	// connectionPool will call this function when it needs a new connection
	newConn   func() *tsConn
	count     int64
	max       int64
	waitTimer *ExponentialTimer
}

func NewConnectionPool(newConn func() *tsConn, maxConnections int) (*connectionPool, error) {
	pool := &connectionPool{newConn: newConn, pool: make(chan *tsConn, maxConnections), count: 0, max: int64(maxConnections), waitTimer: NewExponentialTimer(600)}
	for i := 0; i < maxConnections/2; i++ { // initialize half of the connections
		conn := newConn()
		if conn != nil {
			pool.pool <- conn
		} else {
			return nil, fmt.Errorf("Failed to create connection pool. New connection failed!")
		}
		atomic.AddInt64(&pool.count, 1)
	}
	return pool, nil
}

func (pool *connectionPool) Get() *tsConn {
	var c *tsConn
	select {
	case c = <-pool.pool:
		if c.IsClosed() {
			return pool.Get()
		}
	default:
		if atomic.LoadInt64(&pool.count) < pool.max {
			for {
				c = pool.newConn()
				if c != nil {
					break
				}
				pool.waitTimer.Wait(true)
			}
			atomic.AddInt64(&pool.count, 1)
			log.Info("Creating new connection in pool %v", c.conn, pool.count)
		}
	}
	return c
}

func (pool *connectionPool) Put(c *tsConn) {
	if c.IsClosed() {
		atomic.AddInt64(&pool.count, -1)
		return
	}
	select {
	case pool.pool <- c:
	default:
		c.Close()
		atomic.AddInt64(&pool.count, -1)
		log.Info("Releasing connection in pool, now %v", pool.count)
	}
}
