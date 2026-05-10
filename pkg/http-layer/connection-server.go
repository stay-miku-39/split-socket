package httplayer

// todo: client to server的close帧
// 需要包装http.Request，使其可以关闭（发送信号到阻塞的 handler goroutine）
// timeout全靠connection自身管理

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/stay-miku-39/split-socket/pkg/utils"
)

type HTTPServerConnection struct {
	sessionId SessionId
	connType  ConnectType
	seq       uint16
	useHttp2  bool
	// use for Server to Client
	writer             http.ResponseWriter
	disconnect         bool
	disconnectNotifyer func()
	// use for Client to Server
	lastActivateTimestamp uint64
	windowSize            uint16
	window                *utils.SlideWindow
	readCache             *bytes.Buffer
	readErr               error
	closed                bool
	closeCallback         func()
}

func (c *HTTPServerConnection) writeFrame(b []byte) error {
	flusher, ok := c.writer.(http.Flusher)
	if !ok {
		return fmt.Errorf("Unsupported underlayer connection")
	}
	seq := make([]byte, 2)
	binary.BigEndian.PutUint16(seq, c.seq)
	_, err := c.writer.Write(seq)
	if err != nil {
		return err
	}
	binary.BigEndian.PutUint16(seq, uint16(len(b)))
	_, err = c.writer.Write(seq)
	if err != nil {
		return err
	}
	_, err = c.writer.Write(b)
	if err != nil {
		return err
	}
	flusher.Flush()
	c.seq++
	return nil
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

func (c *HTTPServerConnection) proccessHTTPRequest(writer http.ResponseWriter, request *http.Request) error {
	t := make([]byte, 2)
	_, err := request.Body.Read(t)
	if err != nil {
		return err
	}
	seq := binary.BigEndian.Uint16(t)
	_, err = request.Body.Read(t)
	if err != nil {
		return err
	}
	length := binary.BigEndian.Uint16(t)
	if length > MaxFrameLength {
		return fmt.Errorf("FrameSize exceed max frame size: %v", length)
	}
	err = c.window.Put(seq, func(b []byte) (uint16, error) {
		_, err := io.ReadFull(request.Body, b[:length])
		if err != nil {
			c.readErr = err
			return 0, err
		}
		// todo: 成功响应
		return length, nil
	})
	return err
}

func (c *HTTPServerConnection) setDisconnectNotifyer(notifyer func()) {
	c.disconnectNotifyer = notifyer
}

func (c *HTTPServerConnection) setCloseCallback(callback func()) {
	c.closeCallback = callback
}

func (c *HTTPServerConnection) checkTimeout() {

}
