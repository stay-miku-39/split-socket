package httplayer

// todo: client to server的close帧
// 需要包装http.Request，使其可以关闭（发送信号到阻塞的 handler goroutine）
// timeout全靠connection自身管理

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/stay-miku-39/split-socket/pkg/utils"
)

type HTTPServerConnection struct {
	sessionId SessionId
	connType  ConnectType
	seq       uint16
	useHttp2  bool
	// use for Server to Client
	req             *wrapperedRequest
	mu              sync.Mutex
	disconnect      bool
	closeSignal     chan struct{}
	disconnectTimer *utils.Timer
	// use for Client to Server
	lastActivateTimestamp uint64
	windowSize            uint16
	window                *utils.SlideWindow
	readCache             *bytes.Buffer
	readErr               error
	closed                bool
	closeCallback         func()
	timeout               *utils.Timer
}

func (c *HTTPServerConnection) writeFrame(b []byte) error {
	err := c.req.writeDataFrame(c.seq, b)
	if err != nil {
		c.mu.Lock()
		defer c.mu.Unlock()
		if c.disconnectTimer == nil {
			c.disconnectTimer = utils.NewTimer(func() {
				c.Close()
			})
			c.disconnectTimer.Reset(S2CResumeTimeout)
		}
		c.disconnect = true
		return err
	}
	c.seq++
	return nil
}

func (c *HTTPServerConnection) startHeartbeatTask() {
	if c.connType == C2SConnection {
		return
	}
	go func() {
		ticker := time.NewTicker(HeartBeatInterval)
		for {
			select {
			case <-ticker.C:
				c.req.writeHeartbeatFrame()
			case <-c.closeSignal:
				ticker.Stop()
				return
			}
		}
	}()
}

func (c *HTTPServerConnection) startC2STimeout() {
	c.timeout = utils.NewTimer(func() {
		c.Close()
	})
	c.timeout.Reset(C2STimeout)
}

func (c *HTTPServerConnection) Read(b []byte) (int, error) {
	if c.readErr != nil {
		return 0, c.readErr
	}
	if c.readCache.Len() == 0 {
		if c.connType == S2CConnection {
			return 0, io.EOF
		}
		err := c.window.Get(func(bp []byte) error {
			c.readCache.Write(bp)
			return nil
		})
		if err != nil {
			return 0, err
		}
	}
	return c.readCache.Read(b)
}

func (c *HTTPServerConnection) Write(b []byte) (int, error) {
	if c.connType == C2SConnection {
		return 0, fmt.Errorf("Unsupported operation on this connection")
	}
	if c.closed {
		return 0, io.EOF
	}
	// wrote length
	wl := 0
	for {
		if len(b) > MaxFrameLength {
			err := c.writeFrame(b[:MaxFrameLength])
			if err != nil {
				return wl, err
			}
			b = b[MaxFrameLength:]
			wl += MaxFrameLength
		} else {
			err := c.writeFrame(b)
			if err != nil {
				return wl, err
			}
			wl += len(b)
			break
		}
	}
	return wl, nil
}

func (c *HTTPServerConnection) Close() error {
	if c.closed {
		return nil
	}
	c.closed = true
	c.closeCallback()
	c.window.Close()
	close(c.closeSignal)
	return nil
}

func (c *HTTPServerConnection) LocalAddr() net.Addr {
	return nil
}

func (c *HTTPServerConnection) RemoteAddr() net.Addr {
	return nil
}

func (c *HTTPServerConnection) SetDeadline(t time.Time) error {
	return nil
}

func (c *HTTPServerConnection) SetReadDeadline(t time.Time) error {
	return nil
}

func (c *HTTPServerConnection) SetWriteDeadline(t time.Time) error {
	return nil
}

func (c *HTTPServerConnection) proccessHTTPRequest(r *wrapperedRequest) error {
	if c.connType == S2CConnection {
		c.mu.Lock()
		defer c.mu.Unlock()
		if c.disconnectTimer != nil && c.disconnectTimer.HasFired() {
			return errors.New("This connection is timeout")
		}
		seqString := r.w.Header().Get(ResumeConnectionSeqHeader)
		seq, err := strconv.Atoi(seqString)
		if err != nil {
			return err
		}
		if seq > 255 || seq < 0 {
			return errors.New("Error seq range")
		}
		if uint16(seq) != c.seq {
			return errors.New("Error seq with local seq")
		}
		if c.disconnectTimer != nil {
			c.disconnectTimer.Close()
			c.disconnectTimer = nil
		}
		c.disconnect = false
		c.req = r
		return nil
	} else if c.connType == C2SConnection {
		reqType, err := r.getFrameType()
		if err != nil {
			r.closeWithError(err.Error())
			return err
		}
		if reqType != DataFrameType {
			c.timeout.Reset(C2STimeout)
			return nil
		}
		seq, err := r.getFrameSeq()
		if err != nil {
			r.closeWithError(err.Error())
			return err
		}
		err = c.window.Put(seq, func(b []byte) (uint16, error) {
			_, length, err := r.readFrame(b)
			if err != nil {
				c.readErr = err
				return 0, err
			}
			r.closeWithSuccess("")
			c.timeout.Reset(C2STimeout)
			return length, nil
		})
		return err
	}
	return nil
}

func (c *HTTPServerConnection) setCloseCallback(callback func()) {
	c.closeCallback = callback
}
