package splitlayer

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/stay-miku-39/split-socket/pkg/utils"
)

var serverlogger = utils.NewLogger("split-layer/server")

// addr: addr1,addr2,addr3...
func (t *SplitTransport) Listen(addr string) (net.Listener, error) {
	addrs := strings.Split(addr, ",")
	underlayerListeners := make([]net.Listener, len(addrs))
	for i, a := range addrs {
		listener, err := t.underlayer.Listen(a)
		if err != nil {
			return nil, err
		}
		underlayerListeners[i] = listener
	}
	queue := utils.NewChainList[*SplitConn]()
	pendingMap := make(map[ConnectionId]*SplitConn)
	ctx, cancel := context.WithCancel(context.Background())
	completeQueue := make(chan *SplitConn, t.maxFullConnectionCount)
	underlayerQueue := make(chan net.Conn, t.maxUnderLayerConnectionCount)

	listener := &SplitListener{
		upDownConnectTimeout:         t.upDownConnectTimeout,
		readTimeout:                  t.readTimeout,
		writeTimeout:                 t.writeTimeout,
		underlayer:                   underlayerListeners,
		pendingQueue:                 queue,
		pendingMap:                   pendingMap,
		completeQueue:                completeQueue,
		ctx:                          ctx,
		stop:                         cancel,
		maxHalfConnectionCount:       t.maxHalfConnectionCount,
		maxFullConnectionCount:       t.maxFullConnectionCount,
		maxUnderLayerConnectionCount: t.maxUnderLayerConnectionCount,
		underlayerConnQueue:          underlayerQueue,
	}
	listener.start()
	return listener, nil
}

func (l *SplitListener) start() {
	go func() {
		serverlogger.Debug("start timeout half connection routine")
		ticker := time.NewTicker(time.Second * 1)
		defer ticker.Stop()
		for true {
			select {
			case <-l.ctx.Done(): // exit goroutine
				return
			case <-ticker.C:
				l.checkTimeoutHalfConnection()
			}
		}
	}()

	go func() {
		serverlogger.Debug("start new connection process routine")
		for true {
			select {
			case conn := <-l.underlayerConnQueue:
				l.getNewConnection(conn)
			case <-l.ctx.Done():
				return
			}
		}
	}()

	for _, listener := range l.underlayer {
		go func() {
			serverlogger.Debug("start listener accept routine")
			for true {
				conn, err := listener.Accept()
				if err != nil {
					return
				}
				l.underlayerConnQueue <- conn
			}
		}()
	}
}

// clean timeout half connection
func (l *SplitListener) checkTimeoutHalfConnection() {
	serverlogger.Debug("start clean timeout half connection")
	l.mu.Lock()
	defer l.mu.Unlock()
	i := 0
	defer func() {
		serverlogger.Debug("clean " + strconv.Itoa(i) + " half connection")
	}()
	for true {
		if l.pendingQueue.Len() == 0 {
			return
		}
		conn := l.pendingQueue.At(0)
		if conn.halfTimeoutTimestamp <= time.Now().UnixMilli() {
			conn.Close()
			l.pendingQueue.Pop()
			delete(l.pendingMap, conn.connectionId)
			i++
		} else {
			return
		}
	}
}

func (l *SplitListener) getNewConnection(conn net.Conn) {
	if l.pendingQueue.Len() >= uint(l.maxHalfConnectionCount) || len(l.completeQueue) >= l.maxFullConnectionCount {
		serverlogger.Warn("Reach the max connection count")
		conn.Close()
		return
	}
	firstFrame, err := readFrame(conn, l.readTimeout)
	if err != nil {
		serverlogger.Warn("Error connection: %v", err)
		conn.Close()
		return
	}
	if firstFrame.frameType != ConnectFrame {
		serverlogger.Warn("Error first frame")
		conn.Close()
		return
	}

	var id ConnectionId
	copy(id[:], firstFrame.data)

	splitConn, ok := l.pendingMap[id]
	if !ok {
		serverlogger.Debug("new half connection: %v", id)
		half := l.newHalfConnection(conn, firstFrame.connType, id)
		l.mu.Lock()
		l.pendingMap[id] = half
		l.pendingQueue.Append(half)
		l.mu.Unlock()
	} else {
		if (splitConn.writeConnection != nil && firstFrame.connType == S2CConnect) || (splitConn.readConnection != nil && firstFrame.connType == C2SConnect) {
			serverlogger.Warn("Duplicate connect type with id: %v", id)
			conn.Close()
			return
		}
		serverlogger.Debug("full connection established: %v", id)
		l.mu.Lock()
		for i, c := range l.pendingQueue.Values() {
			if c.connectionId == id {
				l.pendingQueue.Delete(i)
				break
			}
		}
		delete(l.pendingMap, id)
		l.mu.Unlock()
		splitConn.half = false
		if firstFrame.connType == S2CConnect {
			splitConn.writeConnection = conn
		} else {
			splitConn.readConnection = conn
		}
		l.completeQueue <- splitConn
	}
}

func (l *SplitListener) Accept() (net.Conn, error) {
	select {
	case conn := <-l.completeQueue:
		return conn, nil
	case <-l.ctx.Done():
		return nil, fmt.Errorf("Listener has been closed")
	}
}

func (l *SplitListener) Close() error {
	if l.ctx.Err() != nil {
		return nil
	}
	l.stop()
	var rt error
	for _, listener := range l.underlayer {
		err := listener.Close()
		if err != nil {
			rt = fmt.Errorf("One or more error encountered when close the listener")
		}
	}
	for _, conn := range l.pendingQueue.Values() {
		err := conn.Close()
		if err != nil {
			rt = fmt.Errorf("One or more error encountered when close the listener")
		}
	}
	close(l.completeQueue)
	close(l.underlayerConnQueue)
	for conn := range l.completeQueue {
		err := conn.Close()
		if err != nil {
			rt = fmt.Errorf("One or more error encountered when close the listener")
		}
	}
	for conn := range l.underlayerConnQueue {
		err := conn.Close()
		if err != nil {
			rt = fmt.Errorf("One or more error encountered when close the listener")
		}
	}
	return rt
}

func (l *SplitListener) Addr() net.Addr {
	return &SplitAddr{}
}
