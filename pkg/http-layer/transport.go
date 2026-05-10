package httplayer

import splitlayer "github.com/stay-miku-39/split-socket/pkg/split-layer"

type ConnectType int

const MaxFrameLength = 32764

type HTTPTransport struct {
	underlayer splitlayer.Transport
	useHTTP2   bool

	//client
	connectType ConnectType
}

const (
	C2SConnection ConnectType = iota
	S2CConnection
)
