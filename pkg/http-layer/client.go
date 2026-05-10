package httplayer

import "net"

func (t *HTTPTransport) Dial(addr string) (net.Conn, error) {
	return nil, nil
}
