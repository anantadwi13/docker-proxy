package proxy

import (
	"errors"
	tcpproxy "github.com/jpillora/go-tcp-proxy"
	"net"
	"sync"
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

	t.wgConn.Wait()
	return nil
}
