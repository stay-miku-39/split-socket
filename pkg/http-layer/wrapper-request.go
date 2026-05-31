package httplayer

import (
	"encoding/binary"
	"errors"
	"io"
	"net/http"
	"sync"
	"unsafe"

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
	ErrorFrameType   = 254
	UnKnownFrameType = 255
)

func validateFrameType(i uint8) bool {
	return i == uint8(CloseFrameType) || i == uint8(HeartbeatFrameType) || i == uint8(DataFrameType)
}

type wrapperedRequest struct {
	w             http.ResponseWriter
	r             *http.Request
	c             chan struct{}
	t             FrameType
	seq           uint16
	seqProccessed bool
	chunked       bool
	cache         *utils.BufferPool
	mu            sync.Mutex
}

func newWrapperedRequest(writer http.ResponseWriter, request *http.Request) (*wrapperedRequest, chan struct{}) {
	channel := make(chan struct{})
	return &wrapperedRequest{
		w:       writer,
		r:       request,
		c:       channel,
		t:       UnKnownFrameType,
		chunked: false,
		cache:   nil,
		mu:      sync.Mutex{},
	}, channel
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

func (r *wrapperedRequest) closeWithError(e string) (i int, err error) {
	if r.chunked {
		r.close()
		return
	}
	r.notifyError()
	i, err = r.w.Write(unsafe.Slice(unsafe.StringData(e), len(e)))
	r.close()
	return
}

func (r *wrapperedRequest) closeWithSuccess(m string) (i int, err error) {
	if r.chunked {
		r.close()
		return
	}
	r.notifySuccsess()
	i, err = r.w.Write(unsafe.Slice(unsafe.StringData(m), len(m)))
	r.close()
	return
}

func (r *wrapperedRequest) notifySuccsess() {
	r.w.WriteHeader(200)
}

func (r *wrapperedRequest) notifyError() {
	r.w.WriteHeader(400)
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
			return errors.ErrUnsupported
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

func (r *wrapperedRequest) writeDataFrame(seq uint16, d []byte) error {
	return r.writeFrame(d, DataFrameType, seq)
}

func (r *wrapperedRequest) enableChunkedMode() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.chunked = true
	r.cache = utils.NewBufferPool(MaxReserveFrameCount, MaxFrameLength)
	r.w.Header().Set("Content-Type", "application/octet-stream")
}

func (r *wrapperedRequest) getFrameType() (FrameType, error) {
	if r.t != UnKnownFrameType {
		return r.t, nil
	} else if r.t == ErrorFrameType {
		return ErrorFrameType, errors.ErrUnsupported
	}

	t := make([]byte, 1)
	_, err := io.ReadFull(r.r.Body, t)
	if err != nil {
		r.t = ErrorFrameType
		return ErrorFrameType, err
	}
	if validateFrameType(t[0]) {
		r.t = FrameType(t[0])
		return r.t, nil
	} else {
		return ErrorFrameType, errors.ErrUnsupported
	}
}

func (r *wrapperedRequest) getFrameSeq() (uint16, error) {
	t, err := r.getFrameType()
	if err != nil {
		return 0, err
	}
	if t == CloseFrameType || t == HeartbeatFrameType {
		return 0, nil
	}
	d := make([]byte, 2)
	_, err = io.ReadFull(r.r.Body, d)
	if err != nil {
		return 0, err
	}
	seq := binary.BigEndian.Uint16(d)
	return seq, nil
}

// return seq length error
func (r *wrapperedRequest) readFrame(b []byte) (uint16, uint16, error) {
	t, err := r.getFrameType()
	if err != nil {
		return 0, 0, err
	}
	if t == CloseFrameType || t == HeartbeatFrameType {
		return 0, 0, nil
	}
	seq, err := r.getFrameSeq()
	if err != nil {
		return 0, 0, err
	}
	d := make([]byte, 2)
	_, err = io.ReadFull(r.r.Body, d)
	if err != nil {
		return 0, 0, err
	}
	length := binary.BigEndian.Uint16(d)
	if length > MaxFrameLength {
		return 0, 0, errors.New("Data length exceed the max frame length")
	}
	if int(length) > len(b) {
		return 0, 0, errors.New("Data length exceed the max frame length")
	}
	_, err = io.ReadFull(r.r.Body, b[:length])
	return seq, length, nil
}
