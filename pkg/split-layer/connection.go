package splitlayer

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/stay-miku-39/split-socket/pkg/utils"
)

type Flusher interface {
	Flush() error
}

// 128 bit
type ConnectionId [16]byte
type FrameType uint8
type ConnectType uint8

const (
	ConnectFrame FrameType = iota
	DataFrame
)

const (
	C2SConnect ConnectType = iota
	S2CConnect
	EmptyConnect ConnectType = 255
)

type SplitConn struct {
	connectionId            ConnectionId
	half                    bool
	connectTimestamp        int64
	halfTimeoutTimestamp    int64
	writeConnection         net.Conn
	readConnection          net.Conn
	readTimeout             time.Time
	writeTimeout            time.Time
	closed                  bool
	readBuffer              bytes.Buffer
	writeBuffer             bytes.Buffer
	enableWriteCache        bool
	writeCacheFlushInterval time.Duration
	writeCacheMinSendSize   int
	delayWriteMutex         sync.Mutex
}

type SplitFrame struct {
	frameType FrameType   // 0: establish connection 1: data frame
	connType  ConnectType // 0: client->server 1: client<-server, availabe on frameType=0
	len       uint16      // max 16384, 16 on frameType=0
	data      []byte      // frame_type=0: connectionId
}

var connLogger = utils.NewLogger("split-layer/connection")

func readFrameHead(conn net.Conn) (frameType FrameType, connectType ConnectType, length uint16, err error) {
	type_ := make([]byte, 1)
	_, err = io.ReadFull(conn, type_)
	if err != nil {
		return ConnectFrame, EmptyConnect, 0, err
	}
	if type_[0] == byte(ConnectFrame) {
		frameType = ConnectFrame
		_, err = io.ReadFull(conn, type_)
		if err != nil {
			return ConnectFrame, EmptyConnect, 0, err
		}
		if type_[0] != byte(C2SConnect) && type_[0] != byte(S2CConnect) {
			return ConnectFrame, EmptyConnect, 0, fmt.Errorf("unknown connect type: %v", type_[0])
		}
		connectType = ConnectType(type_[0])
	} else if type_[0] == byte(DataFrame) {
		frameType = DataFrame
		connectType = EmptyConnect
	} else {
		return ConnectFrame, EmptyConnect, 0, fmt.Errorf("unknown frame type: %v", type_[0])
	}
	err = binary.Read(conn, binary.BigEndian, &length)
	if err != nil {
		return ConnectFrame, EmptyConnect, 0, err
	}
	if frameType == ConnectFrame && length != 16 {
		return ConnectFrame, EmptyConnect, 0, fmt.Errorf("error data length(ConnectFrame): %v", length)
	}
	if length > 16384 {
		return ConnectFrame, EmptyConnect, 0, fmt.Errorf("error data length: %v", length)
	}
	return
}

func sendFrameHead(conn net.Conn, frameType FrameType, connectType ConnectType, length uint16) error {
	type_ := make([]byte, 1)
	type_[0] = byte(frameType)
	_, err := conn.Write(type_)
	if err != nil {
		return err
	}
	if frameType == ConnectFrame {
		type_[0] = byte(connectType)
		_, err := conn.Write(type_)
		if err != nil {
			return err
		}
	}
	return binary.Write(conn, binary.BigEndian, length)
}

func readFrame(conn net.Conn, timeout time.Duration) (*SplitFrame, error) {
	conn.SetDeadline(time.Now().Add(timeout))
	frame := &SplitFrame{}
	frameType, connType, length, err := readFrameHead(conn)
	if err != nil {
		return nil, err
	}
	frame.frameType = frameType
	frame.connType = connType
	frame.len = length
	frame.data = make([]byte, frame.len)
	_, err = io.ReadFull(conn, frame.data)
	conn.SetDeadline(time.Time{})
	return frame, err
}

func sendFrame(conn net.Conn, frameType FrameType, connType ConnectType, data []byte, timeout time.Duration) error {
	if len(data) >= 16384 {
		return fmt.Errorf("Data too long")
	}
	err := sendFrameHead(conn, frameType, connType, uint16(len(data)))
	if err != nil {
		return err
	}
	_, err = conn.Write(data)
	return err
}

func (c *SplitConn) readFrame() error {
	frameType, _, length, err := readFrameHead(c.readConnection)
	if err != nil {
		return err
	}
	if frameType != DataFrame {
		return fmt.Errorf("Error frame type")
	}
	_, err = io.CopyN(&c.readBuffer, c.readConnection, int64(length))
	return err
}

func (c *SplitConn) writeFrame() error {
	if c.writeBuffer.Len() == 0 {
		return nil
	}
	err := sendFrameHead(c.writeConnection, DataFrame, EmptyConnect, uint16(c.writeBuffer.Len()))
	if err != nil {
		return err
	}
	_, err = c.writeBuffer.WriteTo(c.writeConnection)
	return err
}

func (c *SplitConn) Read(b []byte) (int, error) {
	connLogger.Debug("Read")
	if c.closed {
		return 0, io.EOF
	}
	if c.readBuffer.Len() == 0 {
		err := c.readFrame()
		if err != nil {
			return 0, err
		}
	}
	return c.readBuffer.Read(b)
}

func (c *SplitConn) Write(b []byte) (int, error) {
	connLogger.Debug("Write")
	if c.closed {
		return 0, io.EOF
	}
	writed := 0
	for true {
		if len(b)+c.writeBuffer.Len() > 16384 {
			length := 16384 - c.writeBuffer.Len()
			_, err := c.writeBuffer.Write(b[:length])
			if err != nil {
				return writed, err
			}
			err = c.Flush()
			if err != nil {
				return writed, err
			}
			b = b[length:]
			writed += length
		} else {
			_, err := c.writeBuffer.Write(b)
			if err != nil {
				return writed, err
			}
			if !c.enableWriteCache {
				err = c.Flush()
				if err != nil {
					return writed, err
				}
			} else if c.enableWriteCache && c.writeBuffer.Len() > c.writeCacheMinSendSize {
				err = c.Flush()
				if err != nil {
					return writed, err
				}
			} else if c.enableWriteCache {
				go func() {
					lock := c.delayWriteMutex.TryLock()
					if !lock {
						return
					}
					defer c.delayWriteMutex.Unlock()
					time.Sleep(c.writeCacheFlushInterval)
					err := c.Flush()
					if err != nil {
						connLogger.Warn("auto flush conn faild: %v", c.connectionId)
					}
				}()
			}
			return writed + len(b), nil
		}
	}
	return writed, nil
}

func (c *SplitConn) Flush() error {
	connLogger.Debug("Flush")
	return c.writeFrame()
}

func (c *SplitConn) Close() error {
	connLogger.Debug("Close")
	var rt error
	c.closed = true
	if c.writeConnection != nil {
		c.Flush()
		err := c.writeConnection.Close()
		if err != nil {
			rt = err
		}
	}
	if c.readConnection != nil {
		err := c.readConnection.Close()
		if err != nil {
			if rt == nil {
				rt = err
			} else {
				rt = fmt.Errorf("err1: %v, err2: %v", rt, err)
			}
		}
	}
	return rt
}

func (c *SplitConn) LocalAddr() net.Addr {
	return nil
}

func (c *SplitConn) RemoteAddr() net.Addr {
	return nil
}

func (c *SplitConn) SetDeadline(t time.Time) error {
	c.SetReadDeadline(t)
	c.SetWriteDeadline(t)
	return nil
}

func (c *SplitConn) SetReadDeadline(t time.Time) error {
	c.readTimeout = t
	return c.readConnection.SetReadDeadline(t)
}

func (c *SplitConn) SetWriteDeadline(t time.Time) error {
	c.writeTimeout = t
	return c.writeConnection.SetWriteDeadline(t)
}

func Flush(conn any) error {
	flusher, ok := conn.(Flusher)
	if ok {
		return flusher.Flush()
	}
	return nil
}

func (c *SplitConn) SetEnableWriteCache(enable bool) {
	c.enableWriteCache = enable
}

func (c *SplitConn) GetEnableWriteCache() bool {
	return c.enableWriteCache
}
