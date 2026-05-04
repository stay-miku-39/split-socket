package splitlayer

import (
	"net"
	"time"
)

type Transport interface {
	Dial(addr string) (net.Conn, error)
	Listen(addr string) (net.Listener, error)
	WithTransport(transport Transport)
}

type SplitTransport struct {
	underlayer                   Transport
	upDownConnectTimeout         time.Duration
	readTimeout                  time.Duration
	writeTimeout                 time.Duration
	maxHalfConnectionCount       int
	maxFullConnectionCount       int
	maxUnderLayerConnectionCount int

	// connection
	enableWriteCache        bool
	writeCacheFlushInterval time.Duration
	writeCacheMinSendSize   int
}

type SplitConfig struct {
	// server config
	HalfConnectTimeout           time.Duration
	ReadTimeout                  time.Duration
	WriteTimeout                 time.Duration
	MaxHalfConnectionCount       int
	MaxFullConnectionCount       int
	MaxUnderLayerConnectionCount int

	//conn config
	EnableWriteCache        bool
	WriteCacheFlushInterval time.Duration
	WriteCacheMinSendSize   int
}

func NewSplitTransport(config *SplitConfig) *SplitTransport {
	return &SplitTransport{
		upDownConnectTimeout:         config.HalfConnectTimeout,
		readTimeout:                  config.ReadTimeout,
		writeTimeout:                 config.WriteTimeout,
		maxHalfConnectionCount:       config.MaxHalfConnectionCount,
		maxFullConnectionCount:       config.MaxFullConnectionCount,
		maxUnderLayerConnectionCount: config.MaxUnderLayerConnectionCount,

		enableWriteCache:        config.EnableWriteCache,
		writeCacheFlushInterval: config.WriteCacheFlushInterval,
		writeCacheMinSendSize:   config.WriteCacheMinSendSize,
	}
}

func (t *SplitTransport) WithTransport(transport Transport) {
	t.underlayer = transport
}
