package utils

import (
	"bytes"
	"io"
	"sync"
)

type buffer struct {
	data bytes.Buffer
	meta int
}

type BufferPool struct {
	cap   int
	len   int
	start int
	data  []*buffer
	mu    sync.Mutex
}

func NewBufferPool(size int, initBufferSlotSize int) *BufferPool {
	if size <= 0 || initBufferSlotSize <= 0 {
		return nil
	}
	data := make([]*buffer, size)
	for i := 0; i < size; i++ {
		data[i] = &buffer{
			meta: 0,
			data: *bytes.NewBuffer(make([]byte, 0, initBufferSlotSize)),
		}
	}
	return &BufferPool{
		len:   0,
		cap:   size,
		start: 0,
		data:  data,
		mu:    sync.Mutex{},
	}
}

func (b *BufferPool) WriteBuffer(callback func(*bytes.Buffer) int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.len >= b.cap {
		buf := b.data[b.start]
		buf.data.Reset()
		b.start = (b.start + 1) % b.cap
		buf.meta = callback(&buf.data)
		return
	} else {
		buf := b.data[(b.start+b.len)%b.cap]
		buf.data.Reset()
		b.len++
		buf.meta = callback(&buf.data)
		return
	}
}

func (b *BufferPool) ReadBuffer(callback func(*bytes.Buffer, int)) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.len <= 0 {
		return io.EOF
	} else {
		buf := b.data[b.start]
		b.start = (b.start + 1) % b.cap
		b.len--
		callback(&buf.data, buf.meta)
		return nil
	}
}
