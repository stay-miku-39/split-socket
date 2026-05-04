package splitlayer

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/stay-miku-39/split-socket/pkg/utils"
)

type SplitListener struct {
	ctx                          context.Context
	stop                         func()
	underlayer                   []net.Listener
	upDownConnectTimeout         time.Duration // minimal 1s
	readTimeout                  time.Duration
	writeTimeout                 time.Duration
	maxHalfConnectionCount       int
	maxFullConnectionCount       int
	maxUnderLayerConnectionCount int

	// mutex protected
	pendingQueue *utils.ChainList[*SplitConn] // use for upDownConnectTimeout, half connection
	pendingMap   map[ConnectionId]*SplitConn  // use for find connection, half connection

	completeQueue chan *SplitConn

	underlayerConnQueue chan net.Conn

	mu sync.Mutex
}

type SplitAddr struct {
	network string
	str     string
}

func (a *SplitAddr) Network() string {
	return a.network
}

func (a *SplitAddr) String() string {
	return a.str
}

func (l *SplitListener) newHalfConnection(conn net.Conn, connType ConnectType, connId ConnectionId) *SplitConn {
	var up, down net.Conn
	if connType == C2SConnect {
		down = conn
	} else {
		up = conn
	}
	return &SplitConn{
		connectionId:         connId,
		half:                 true,
		closed:               false,
		connectTimestamp:     time.Now().UnixMilli(),
		halfTimeoutTimestamp: time.Now().Add(l.upDownConnectTimeout).UnixMilli(),
		writeConnection:      up,
		readConnection:       down,
		readTimeout:          time.Time{},
		writeTimeout:         time.Time{},
	}
}
