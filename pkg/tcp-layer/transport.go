package tcplayer

import (
	"net"

	splitlayer "github.com/stay-miku-39/split-socket/pkg/split-layer"
)

type TCPTransport struct {
}

func (t *TCPTransport) Dial(addr string) (net.Conn, error) {
	return net.Dial("tcp", addr)
}

func (t *TCPTransport) Listen(addr string) (net.Listener, error) {
	return net.Listen("tcp", addr)
}

func (t *TCPTransport) WithTransport(transport splitlayer.Transport) {
}
