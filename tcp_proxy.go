package main

import (
	"errors"
	"fmt"
	tcpproxy "github.com/jpillora/go-tcp-proxy"
	"net"
	"sync"
	"time"
)

type tcpProxy struct {
	localAddr  *net.TCPAddr
	remoteAddr *net.TCPAddr

	isClosed bool
	rwMu     sync.RWMutex
	wgConn   sync.WaitGroup
}

func NewTcpProxy(laddr, raddr *net.TCPAddr) (Proxy, error) {
	if laddr == nil || raddr == nil {
		return nil, errors.New("you need to specify both local address and remote address")
	}
	return &tcpProxy{localAddr: laddr, remoteAddr: raddr}, nil
}

func (t *tcpProxy) Start() error {
	listener, err := net.ListenTCP("tcp", t.localAddr)
	if err != nil {
		panic(err)
	}
	fmt.Printf("listening tcp proxy on %v with remote address %v\n", t.localAddr.String(), t.remoteAddr.String())
	for {
		t.rwMu.RLock()
		if t.isClosed {
			t.rwMu.RUnlock()
			listener.Close()
			return nil
		}
		t.rwMu.RUnlock()
		lconn, err := listener.AcceptTCP()
		if err != nil {
			panic(err)
		}
		p := tcpproxy.New(lconn, t.localAddr, t.remoteAddr)
		go func() {
			t.wgConn.Add(1)
			p.Start()
			t.wgConn.Done()
		}()
	}
}

func (t *tcpProxy) Shutdown() error {
	t.rwMu.Lock()
	t.isClosed = true
	t.rwMu.Unlock()

	done := make(chan interface{})

	go func() {
		defer close(done)
		t.wgConn.Wait()
	}()

	select {
	case <-time.After(10 * time.Second):
		fmt.Println("force shutdown after waiting 10 seconds")
	case <-done:
	}
	return nil
}
