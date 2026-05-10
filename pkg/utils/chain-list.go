package utils

import (
	"fmt"
	"iter"
	"strings"
)

type ChainList[T any] struct {
	first *ChainNode[T]
	last  *ChainNode[T]
	len   uint
}

type ChainNode[T any] struct {
	data T
	head *ChainNode[T]
	end  *ChainNode[T]
}

func NewChainList[T any]() *ChainList[T] {
	return &ChainList[T]{
		first: nil,
		last:  nil,
		len:   0,
	}
}

func newChainNode[T any](data T, head, end *ChainNode[T]) *ChainNode[T] {
	return &ChainNode[T]{
		data: data,
		head: head,
		end:  end,
	}
}

// queue method
func (c *ChainList[T]) Append(item T) {
	if c.len == 0 {
		newNode := newChainNode(item, nil, nil)
		c.first = newNode
		c.last = newNode
		c.len++
		return
	}
	newNode := newChainNode(item, c.last, nil)
	c.last.end = newNode
	c.last = newNode
	c.len++
}

// stack method
func (c *ChainList[T]) Push(item T) {
	if c.len == 0 {
		newNode := newChainNode(item, nil, nil)
		c.first = newNode
		c.last = newNode
		c.len++
		return
	}
	newNode := newChainNode(item, nil, c.first)
	c.first.head = newNode
	c.first = newNode
	c.len++
}

// queue&stack method
func (c *ChainList[T]) Pop() T {
	if c.len == 0 {
		panic("no more item")
	}
	node := c.first
	c.first = c.first.end
	c.first.head = nil
	c.len--
	return node.data
}

func (c *ChainList[T]) PopEnd() T {
	if c.len == 0 {
		panic("no more item")
	}
	node := c.last
	c.last = c.last.head
	c.last.end = nil
	c.len--
	return node.data
}

func (c *ChainList[T]) nodeAt(index uint) *ChainNode[T] {
	if index >= c.len {
		panic("index out of range")
	}
	item := c.first
	for range index {
		item = item.end
	}
	return item
}

// list method
func (c *ChainList[T]) At(index uint) T {
	return c.nodeAt(index).data
}

// list method
func (c *ChainList[T]) Insert(item T, index uint) {
	if index >= c.len {
		c.Append(item)
		return
	}

	if index == 0 {
		c.Push(item)
		return
	}

	oldNode := c.nodeAt(index)
	oldNodeHead := oldNode.head
	newNode := newChainNode(item, oldNodeHead, oldNode)
	oldNodeHead.end = newNode
	oldNode.head = newNode
	c.len++
}

// list method
func (c *ChainList[T]) Delete(index uint) T {
	if index >= c.len {
		panic("index out of range")
	}
	if c.len == 1 {
		node := c.first
		c.first = nil
		c.last = nil
		c.len = 0
		return node.data
	}
	if index == 0 {
		return c.Pop()
	}
	if index == c.len-1 {
		return c.PopEnd()
	}

	node := c.nodeAt(index)
	node.head.end = node.end
	node.end.head = node.head
	c.len--
	return node.data
}

func (c *ChainList[T]) Len() uint {
	return c.len
}

// for range
func (c *ChainList[T]) Values() iter.Seq2[uint, T] {
	var index uint = 0
	node := c.first
	return func(yield func(uint, T) bool) {
		for node != nil && yield(index, node.data) {
			node = node.end
			index++
		}
	}
}

func (c *ChainList[T]) String() string {
	if c.len == 0 {
		return "ChainList[]{len: 0}"
	}
	if !isBasicType(c.At(0)) && !isStringer(c.At(0)) {
		return fmt.Sprintf("ChainList[Unsupport Type]{len: %v}", c.len)
	}
	var builder strings.Builder

	fmt.Fprintf(&builder, "ChainList[%T]{\n", c.At(0))
	for _, d := range c.Values() {
		fmt.Fprintf(&builder, "  %v,\n", d)
	}
	builder.WriteString("}")

	return builder.String()
}

func (c *ChainList[T]) ForEach(callback func(index uint, item T) bool) {
	node := c.first
	var index uint = 0
	for node != nil && callback(index, node.data) {
		node = node.end
		index++
	}
}

func (c *ChainList[T]) DeleteF(callback func(index uint, item T) bool) (deleted int) {
	if c.len == 0 {
		return
	}
	node := c.first
	var index uint = 0
	deleted = 0
	for node != nil {
		if callback(index, node.data) {
			deleted++
			if node.head == nil && node.end == nil {
				c.first = nil
				c.last = nil
				c.len = 0
				return
			} else if node.head == nil {
				c.len--
				c.first = node.end
				c.first.head = nil
			} else if node.end == nil {
				c.len--
				c.last = node.head
				c.last.end = nil
				return
			} else {
				c.len--
				node.head.end = node.end
				node.end.head = node.head
			}
		}
		node = node.end
		index++
	}
	return
}

func (c *ChainList[T]) DeleteFOnce(callback func(index uint, item T) bool) {
	if c.len == 0 {
		return
	}
	node := c.first
	var index uint = 0
	for node != nil {
		if callback(index, node.data) {
			if node.head == nil && node.end == nil {
				c.first = nil
				c.last = nil
				c.len = 0
			} else if node.head == nil {
				c.len--
				c.first = node.end
				c.first.head = nil
			} else if node.end == nil {
				c.len--
				c.last = node.head
				c.last.end = nil
			} else {
				c.len--
				node.head.end = node.end
				node.end.head = node.head
			}
			return
		}
		node = node.end
		index++
	}
}
