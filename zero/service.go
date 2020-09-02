package zero

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"
)

// SocketService struct
type SocketService struct {
	onMessage    func(*Session, *Message)
	onConnect    func(*Session)
	onDisconnect func(*Session, error)
	sessions     *sync.Map
	hbInterval   time.Duration
	hbTimeout    time.Duration
	laddr        string
	status       int
	listener     net.Listener
	stopCh       chan error
}

// NewSocketService create a new socket service
func NewSocketService(laddr string, hbInterval, hbTimeout time.Duration) (*SocketService, error) {
	l, err := net.Listen("tcp", laddr)
	if err != nil {
		return nil, err
	}
	s := &SocketService{
		sessions:   &sync.Map{},
		stopCh:     make(chan error),
		hbInterval: hbInterval,
		hbTimeout:  hbTimeout,
		laddr:      laddr,
		status:     STInited,
		listener:   l,
	}
	return s, nil
}

// RegMessageHandler register message handler
func (s *SocketService) RegMessageHandler(handler func(*Session, *Message)) {
	s.onMessage = handler
}

// RegConnectHandler register connect handler
func (s *SocketService) RegConnectHandler(handler func(*Session)) {
	s.onConnect = handler
}

// RegDisconnectHandler register disconnect handler
func (s *SocketService) RegDisconnectHandler(handler func(*Session, error)) {
	s.onDisconnect = handler
}

// Serv Start socket service
func (s *SocketService) Run() {
	s.status = STRunning
	ctx, cancel := context.WithCancel(context.Background())

	defer func() {
		s.status = STStop
		cancel()
		s.listener.Close()
	}()
	go s.acceptHandler(ctx)
	for {
		select {
		case <-s.stopCh:
			return
		}
	}
}

func (s *SocketService) acceptHandler(ctx context.Context) {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			s.stopCh <- err
			return
		}
		go s.connectHandler(ctx, conn)
	}
}

func (s *SocketService) connectHandler(ctx context.Context, c net.Conn) {
	conn := NewConn(c, s.hbInterval, s.hbTimeout)
	session := NewSession(conn)
	s.sessions.Store(session.GetSessionID(), session)
	connctx, cancel := context.WithCancel(ctx)

	defer func() {
		cancel()
		conn.Close()
		s.sessions.Delete(session.GetSessionID())
	}()

	go conn.readCoroutine(connctx) //读取网络数据
	go conn.writeCoroutine(connctx)

	if s.onConnect != nil {
		s.onConnect(session)
	}

	for {
		select {
		case err := <-conn.done: //进程关闭
			if s.onDisconnect != nil {
				s.onDisconnect(session, err)
			}
			return

		case msg := <-conn.msgChan:
			fmt.Printf("_________%v\n", msg)
			if s.onMessage != nil {
				s.onMessage(session, msg)
			}
		}
	}
}

// GetStatus get socket service status
func (s *SocketService) GetStatus() int {
	return s.status
}

// Stop stop socket service with reason
func (s *SocketService) Stop(reason string) {
	s.stopCh <- errors.New(reason)
}

// SetHeartBeat set heart beat
func (s *SocketService) SetHeartBeat(hbInterval time.Duration, hbTimeout time.Duration) error {
	if s.status == STRunning {
		return errors.New("Can't set heart beat on service running")
	}
	s.hbInterval = hbInterval
	s.hbTimeout = hbTimeout
	return nil
}

// GetConnsCount get connect count
func (s *SocketService) GetConnsCount() int {
	var count int
	s.sessions.Range(func(k, v interface{}) bool {
		count++
		return true
	})
	return count
}

// Unicast Unicast with session ID
func (s *SocketService) Unicast(sid string, msg *Message) {
	v, ok := s.sessions.Load(sid)
	if ok {
		session := v.(*Session)
		err := session.GetConn().SendMessage(msg)
		if err != nil {
			return
		}
	}
}

// Broadcast Broadcast to all connections
func (s *SocketService) Broadcast(msg *Message) {
	s.sessions.Range(func(k, v interface{}) bool {
		s := v.(*Session)
		if err := s.GetConn().SendMessage(msg); err != nil {
			// log.Println(err)
		}
		return true
	})
}
