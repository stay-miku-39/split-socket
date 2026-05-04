package utils

import (
	"fmt"
	"strings"
)

type Queue[T any] struct {
	data            []T
	len, start, end int
	full            bool
}

func NewQueue[T any](len int) *Queue[T] {
	data := make([]T, len)
	return &Queue[T]{
		data:  data,
		len:   len,
		start: 0,
		end:   0,
		full:  false,
	}
}

func (q *Queue[T]) extend() {
	if !q.full {
		return
	}
	newData := make([]T, q.len*2)
	copy(newData, q.data[q.start:])
	copy(newData[q.len-q.start:], q.data[:q.start])
	q.data = newData
	q.start = 0
	q.end = q.len
	q.len = q.len * 2
	q.full = false
}

func (q *Queue[T]) Append(item T) {
	if q.full {
		q.extend()
	}
	q.data[q.end] = item
	q.end = (q.end + 1) % q.len
	if q.end == q.start {
		q.full = true
	}
}

func (q *Queue[T]) Pop() T {
	if !q.full && q.start == q.end {
		panic("no more item")
	}
	item := q.data[q.start]
	q.start = (q.start + 1) % q.len
	if q.full {
		q.full = false
	}
	return item
}

func (q *Queue[T]) At(index int) T {
	if index >= q.Len() {
		panic("index out of range")
	}
	return q.data[(q.start+index)%q.len]
}

func (q *Queue[T]) Len() int {
	if q.full {
		return q.len
	}
	if q.start > q.end {
		return q.len - (q.start - q.end)
	}
	return q.end - q.start
}

func isBasicType(item any) bool {
	switch item.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, uintptr, float32, float64, complex64, complex128:
		return true
	default:
		return false
	}
}

func isStringer(item any) bool {
	_, ok := item.(fmt.Stringer)
	return ok
}

func (q *Queue[T]) String() string {
	len := q.Len()
	if len == 0 {
		return "Queue[]{len: 0}"
	}
	if !isBasicType(q.At(0)) && !isStringer(q.At(0)) {
		return fmt.Sprintf("Queue[Unsupport Type]{len: %v}", len)
	}
	var rt strings.Builder
	fmt.Fprintf(&rt, "Queue[%T]{\n", q.At(0))
	for i := range len {
		fmt.Fprintf(&rt, "  %v,\n", q.At(i))
	}
	rt.WriteString("}")
	return rt.String()
}
