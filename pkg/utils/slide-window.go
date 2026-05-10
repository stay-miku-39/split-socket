package utils

import (
	"errors"
	"io"
	"strconv"
	"sync"
)

type cache struct {
	used     bool
	complete bool
	len      uint16
	data     []byte
}

func newCache(maxSize uint16) *cache {
	return &cache{
		used:     false,
		data:     make([]byte, maxSize),
		complete: false,
		len:      0,
	}
}

// concurrent safe slide window & cache pool
type SlideWindow struct {
	size    uint16
	baseSeq uint16
	window  []*cache
	mutex   *sync.Mutex
	cond    *sync.Cond
	// completeCallback func(b []byte, len uint16) error
	closed bool
}

func NewSlideWindow(size uint16, maxCacheSize uint16) *SlideWindow {
	window := make([]*cache, size)
	for i := range size {
		window[i] = newCache(maxCacheSize)
	}
	mutex := &sync.Mutex{}
	cond := sync.NewCond(mutex)
	return &SlideWindow{
		size:    size,
		baseSeq: 0,
		window:  window,
		mutex:   mutex,
		cond:    cond,
		// completeCallback: func(b []byte, len uint16) error { return nil },
		closed: false,
	}
}

// func (w *SlideWindow) checkComplete() {
// 	changed := false
// 	w.mutex.Lock()
// 	for {
// 		cache := w.getCache(w.baseSeq)
// 		if cache.used && cache.complete {
// 			w.completeCallback(cache.data, cache.len)
// 			cache.len = 0
// 			cache.used = false
// 			cache.complete = false
// 			w.baseSeq++
// 			changed = true
// 			continue
// 		} else {

// 			break
// 		}
// 	}
// 	w.mutex.Unlock()
// 	if changed {
// 		w.cond.Broadcast()
// 	}
// }

func (w *SlideWindow) Get(callback func(b []byte) error) error {
	w.mutex.Lock()
	var cache *cache
	for {
		cache = w.getCache(w.baseSeq)
		if w.closed && !cache.used {
			return io.EOF
		} else if !(cache.used && cache.complete) {
			w.cond.Wait()
		} else {
			break
		}
	}
	err := callback(cache.data[:cache.len])
	cache.used = false
	cache.complete = false
	cache.len = 0
	w.baseSeq++
	w.mutex.Unlock()
	w.cond.Broadcast()
	return err
}

// func (w *SlideWindow) SetComepleteCallback(callback func(b []byte, len uint16) error) {
// 	w.completeCallback = callback
// }

func checkSeq(seq uint16, baseSeq uint16) int16 {
	return int16(seq - baseSeq)
}

func (w *SlideWindow) getCache(seq uint16) *cache {
	index := seq % w.size
	return w.window[index]
}

func (w *SlideWindow) put_(c *cache, callback func(b []byte) (uint16, error)) {
	if len, err := callback(c.data); err == nil {
		w.mutex.Lock()
		c.len = len
		c.complete = true
		w.mutex.Unlock()
		w.cond.Broadcast()
		// w.checkComplete()
	} else {
		w.mutex.Lock()
		c.used = false
		w.mutex.Unlock()
	}
}

// return when apply cache success
func (w *SlideWindow) PutWithWait(seq uint16, callback func(b []byte) (uint16, error)) error {
	w.mutex.Lock()
	// check seq
	for {
		if w.closed {
			w.mutex.Unlock()
			return errors.New("Apply cache failed, Slide window closed")
		}
		r := checkSeq(seq, w.baseSeq)
		if r < 0 {
			w.mutex.Unlock()
			return errors.New("Apply cache failed, Stale package: " + strconv.Itoa(int(seq)))
		} else if uint16(r) >= w.size {
			// pending when baseSeq move forward
			w.cond.Wait()
		} else {
			break
		}
	}
	cache := w.getCache(seq)
	if cache.used {
		w.mutex.Unlock()
		return errors.New("Apply cache failed, Duplicate package: " + strconv.Itoa(int(seq)))
	}

	cache.used = true
	cache.complete = false
	w.mutex.Unlock()
	go w.put_(cache, callback)
	return nil
}

// return when apply cache success
func (w *SlideWindow) Put(seq uint16, callback func(b []byte) (uint16, error)) error {
	w.mutex.Lock()
	if w.closed {
		w.mutex.Unlock()
		return errors.New("Apply cache failed, Slide window closed")
	}
	// check seq
	r := checkSeq(seq, w.baseSeq)
	if r < 0 {
		w.mutex.Unlock()
		return errors.New("Apply cache failed, Stale package: " + strconv.Itoa(int(seq)))
	} else if uint16(r) >= w.size {
		w.mutex.Unlock()
		return errors.New("Apply cache failed, Seq out of window size: " + strconv.Itoa(int(seq)))
	}
	cache := w.getCache(seq)
	if cache.used {
		w.mutex.Unlock()
		return errors.New("Apply cache failed, Duplicate package: " + strconv.Itoa(int(seq)))
	}

	cache.used = true
	cache.complete = false
	w.mutex.Unlock()
	go w.put_(cache, callback)
	return nil
}

func (w *SlideWindow) Close() {
	w.mutex.Lock()
	if w.closed {
		w.mutex.Unlock()
		return
	}
	w.closed = true
	w.mutex.Unlock()
	w.cond.Broadcast()
}
