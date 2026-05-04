package splitlayer

import (
	"bytes"
	"crypto/rand"
	"net"
	"strings"
	"time"
)

// addr: readAddr,writeAddr  or  addr
func (t *SplitTransport) Dial(addr string) (net.Conn, error) {
	addrs := strings.Split(addr, ",")
	var readAddr, writeAddr string
	if len(addrs) >= 2 {
		readAddr = addrs[0]
		writeAddr = addrs[1]
	} else {
		readAddr = addrs[0]
		writeAddr = addrs[0]
	}
	var connId ConnectionId
	rand.Read(connId[:])

	writeConn, err := t.underlayer.Dial(writeAddr)
	if err != nil {
		return nil, err
	}
	readConn, err := t.underlayer.Dial(readAddr)
	if err != nil {
		return nil, err
	}

	err = sendFrame(writeConn, ConnectFrame, C2SConnect, connId[:], t.writeTimeout)
	if err != nil {
		return nil, err
	}
	err = sendFrame(readConn, ConnectFrame, S2CConnect, connId[:], t.writeTimeout)
	if err != nil {
		return nil, err
	}

	return &SplitConn{
		connectionId:         connId,
		half:                 false,
		connectTimestamp:     time.Now().UnixMilli(),
		halfTimeoutTimestamp: 0,
		writeConnection:      writeConn,
		readConnection:       readConn,
		readTimeout:          time.Time{},
		writeTimeout:         time.Time{},
		closed:               false,
		readBuffer:           bytes.Buffer{},
		writeBuffer:          bytes.Buffer{},
	}, nil
}
