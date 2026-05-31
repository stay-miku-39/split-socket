package httplayer

import (
	"time"

	splitlayer "github.com/stay-miku-39/split-socket/pkg/split-layer"
)

type ConnectType int

const MaxFrameLength = 32764
const HeartBeatInterval = 25 * time.Second
const C2STimeout = 30 * time.Second
const S2CResumeTimeout = 15 * time.Second
const ConnectTypeHeader = "X-Connection-Type"
const ResumeConnectionSeqHeader = "X-Resume-Seq"

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
