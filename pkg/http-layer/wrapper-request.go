package httplayer

import (
	"encoding/binary"
	"errors"
	"io"
	"net/http"
	"sync"

	"github.com/stay-miku-39/split-socket/pkg/utils"
)

const (
	MaxReserveFrameCount = 10
)

type FrameType uint8

const (
	CloseFrameType FrameType = iota
	HeartbeatFrameType
	DataFrameType
)

type wrapperedRequest struct {
	w       http.ResponseWriter
	r       *http.Request
	c       chan int
	chunked bool
	cache   *utils.BufferPool
	mu      sync.Mutex
}

func newWrapperedRequest(writer http.ResponseWriter, request *http.Request) *wrapperedRequest {
	channel := make(chan int)
	return &wrapperedRequest{
		w:       writer,
		r:       request,
		c:       channel,
		chunked: false,
		cache:   nil,
		mu:      sync.Mutex{},
	}
}

func (r *wrapperedRequest) close() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.chunked {
		r.writeCloseFrame()
	}
	io.Copy(io.Discard, r.r.Body)
	r.r.Body.Close()
	close(r.c)
}

// frame header:  type:1byte|seq:2byte|length:2byte
// close frame and heartbeat frame do not use seq field
func (r *wrapperedRequest) writeFrame(b []byte, t FrameType, seq uint16) error {
	if len(b) > MaxFrameLength {
		return errors.New("Data length exceed the max frame length")
	}
	var flusher http.Flusher
	if r.chunked {
		f, ok := r.w.(http.Flusher)
		if !ok {
			return errors.New("Unsupport connection")
		}
		flusher = f
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	_, err := r.w.Write([]byte{byte(t)})
	if err != nil {
		return err
	}

	h := make([]byte, 2)
	binary.BigEndian.PutUint16(h, seq)
	_, err = r.w.Write(h)
	if err != nil {
		return err
	}
	if t == CloseFrameType || t == HeartbeatFrameType {
		binary.BigEndian.PutUint16(h, 0)
		_, err = r.w.Write(h)
		if err != nil {
			return err
		}
	} else if t == DataFrameType {
		binary.BigEndian.PutUint16(h, uint16(len(b)))
		_, err = r.w.Write(h)
		if err != nil {
			return err
		}
	}
	if r.chunked {
		flusher.Flush()
	}
	return nil
}

func (r *wrapperedRequest) writeCloseFrame() error {
	return r.writeFrame(nil, CloseFrameType, 0)
}

// Server to Client condition: just for connection keep-alive on CDN
func (r *wrapperedRequest) writeHeartbeatFrame() error {
	return r.writeFrame(nil, HeartbeatFrameType, 0)
}

func (r *wrapperedRequest) enableChunkedMode() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.chunked = true
	r.cache = utils.NewBufferPool(MaxReserveFrameCount, MaxFrameLength)
	r.w.Header().Set("Content-Type", "application/octet-stream")
}

func (r *wrapperedRequest) readFrame(b []byte) error {

}
