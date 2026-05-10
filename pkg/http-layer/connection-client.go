package httplayer

import (
	"bytes"
	"net"
	"net/http"
	"time"

	"github.com/stay-miku-39/split-socket/pkg/utils"
)

type HTTPClientConnection struct {
	sessionId             SessionId
	connType              ConnectType
	seq                   uint16
	writer                bytes.Buffer
	reader                http.Response
	useHttp2              bool
	client                *http.Client
	lastActivateTimestamp uint64
	window                *utils.SlideWindow
	writeErr              error
}

func (c *HTTPClientConnection) Read(b []byte) (int, error) {
	return 0, nil
}

func (c *HTTPClientConnection) Write(b []byte) (int, error) {
	return 0, nil
}

func (c *HTTPClientConnection) Close() error {
	return nil
}

func (c *HTTPClientConnection) LocalAddr() net.Addr {
	return nil
}

func (c *HTTPClientConnection) RemoteAddr() net.Addr {
	return nil
}

func (c *HTTPClientConnection) SetDeadline(t time.Time) error {
	return nil
}

func (c *HTTPClientConnection) SetReadDeadline(t time.Time) error {
	return nil
}

func (c *HTTPClientConnection) SetWriteDeadline(t time.Time) error {
	return nil
}
