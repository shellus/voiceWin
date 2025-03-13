package capture

import (
	smallnest "github.com/smallnest/ringbuffer"
)

// RingBuffer 环形缓冲区
type RingBuffer struct {
	sn *smallnest.RingBuffer
}

// NewRingBuffer 创建新的环形缓冲区
func NewRingBuffer(size int) *RingBuffer {
	return &RingBuffer{
		sn: smallnest.New(size),
	}
}

// Write 写入数据到环形缓冲区
// 如果数据长度超过缓冲区大小，只保留最后size个字节
// 如果缓冲区空间不足，会先读取一些数据以腾出空间
func (rb *RingBuffer) Write(data []byte) int {
	size := rb.Size()

	// 如果数据长度超过缓冲区大小，只保留最后size个字节
	if len(data) > size {
		data = data[len(data)-size:]
	}

	// 检查剩余空间
	free := rb.Free()
	if free < len(data) {
		// 需要读取的数据量
		needToRead := len(data) - free
		// 读取数据以腾出空间
		rb.Read(needToRead)
	}

	// 现在应该有足够的空间写入数据
	n, _ := rb.sn.Write(data)
	return n
}

// Read 读取数据从环形缓冲区
func (rb *RingBuffer) Read(size int) []byte {
	data := make([]byte, size)
	n, _ := rb.sn.Read(data)
	if n == 0 {
		return nil
	}
	return data[:n]
}

// Available 返回可读取的数据量
func (rb *RingBuffer) Available() int {
	return rb.sn.Length()
}

// Size 返回缓冲区总大小
func (rb *RingBuffer) Size() int {
	return rb.sn.Capacity()
}

// Reset 重置缓冲区
func (rb *RingBuffer) Reset() {
	rb.sn.Reset()
}

// Free 返回可写入的空闲空间
func (rb *RingBuffer) Free() int {
	return rb.sn.Free()
}

// IsEmpty 检查缓冲区是否为空
func (rb *RingBuffer) IsEmpty() bool {
	return rb.sn.IsEmpty()
}

// IsFull 检查缓冲区是否已满
func (rb *RingBuffer) IsFull() bool {
	return rb.sn.IsFull()
}
