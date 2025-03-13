package capture

import (
	"bytes"
	"testing"

	"github.com/smallnest/ringbuffer"
)

func TestSmallnest_Basic(t *testing.T) {
	size := 10
	rb := ringbuffer.New(size)
	if rb.Capacity() != size {
		t.Errorf("期望缓冲区大小为%d，实际为%d", size, rb.Capacity())
	}

	// 测试写入数据
	data := []byte{1, 2, 3, 4, 5}
	n, err := rb.Write(data)
	if err != nil {
		t.Errorf("写入数据失败: %v", err)
	}
	if n != 5 {
		t.Errorf("期望写入5字节，实际写入%d字节", n)
	}

	// 测试可用数据量
	if rb.Length() != 5 {
		t.Errorf("期望可用数据量为5，实际为%d", rb.Length())
	}

	// 测试读取数据
	readData := make([]byte, 3)
	n, err = rb.Read(readData)
	if err != nil {
		t.Errorf("读取数据失败: %v", err)
	}
	if !bytes.Equal(readData[:n], []byte{1, 2, 3}) {
		t.Errorf("读取的数据不匹配，期望[1,2,3]，实际为%v", readData[:n])
	}

	// 测试读取后的可用数据量
	if rb.Length() != 2 {
		t.Errorf("读取后期望可用数据量为2，实际为%d", rb.Length())
	}
}

func TestSmallnest_Overflow(t *testing.T) {
	rb := ringbuffer.New(5)

	// 写入数据直到缓冲区满
	data := []byte{1, 2, 3, 4, 5}
	n, err := rb.Write(data)
	if err != nil {
		t.Errorf("写入数据失败: %v", err)
	}
	if n != 5 {
		t.Errorf("期望写入5字节，实际写入%d字节", n)
	}

	// 方法1：使用TryWrite尝试写入更多数据
	data = []byte{6, 7, 8}
	n, err = rb.TryWrite(data)
	if n != 0 {
		t.Errorf("缓冲区已满时TryWrite应返回0，实际返回%d", n)
	}

	// 方法2：检查剩余空间
	if rb.Free() > 0 {
		_, err = rb.Write(data)
		if err != nil {
			t.Errorf("写入数据失败: %v", err)
		}
	}

	// 方法3：读取一些数据后再写入
	readData := make([]byte, 2)
	n, err = rb.Read(readData)
	if err != nil {
		t.Errorf("读取数据失败: %v", err)
	}
	if !bytes.Equal(readData[:n], []byte{1, 2}) {
		t.Errorf("读取的数据不匹配，期望[1,2]，实际为%v", readData[:n])
	}

	// 现在有空间了，可以写入新数据
	n, err = rb.Write([]byte{6, 7})
	if err != nil {
		t.Errorf("写入数据失败: %v", err)
	}
	if n != 2 {
		t.Errorf("期望写入2字节，实际写入%d字节", n)
	}

	// 验证最终的数据
	readData = make([]byte, 5)
	n, err = rb.Read(readData)
	if err != nil {
		t.Errorf("读取数据失败: %v", err)
	}
	expected := []byte{3, 4, 5, 6, 7}
	if !bytes.Equal(readData[:n], expected) {
		t.Errorf("最终数据不匹配，期望%v，实际为%v", expected, readData[:n])
	}

	// 方法4：重置缓冲区
	rb.Reset()
	if !rb.IsEmpty() {
		t.Error("重置后缓冲区应该为空")
	}
	_, err = rb.Write(data)
	if err != nil {
		t.Errorf("重置后写入数据失败: %v", err)
	}
}

func TestSmallnest_WrapAround(t *testing.T) {
	rb := ringbuffer.New(5)

	// 第一次写入
	rb.Write([]byte{1, 2, 3})

	// 读取部分数据
	buf := make([]byte, 2)
	rb.Read(buf)

	// 再次写入，测试是否正确环绕
	rb.Write([]byte{4, 5, 6})

	// 读取所有可用数据
	data := make([]byte, 4)
	n, _ := rb.Read(data)
	expected := []byte{3, 4, 5, 6}
	if !bytes.Equal(data[:n], expected) {
		t.Errorf("环绕后读取的数据不匹配，期望%v，实际为%v", expected, data[:n])
	}
}

func TestSmallnest_ReadEmpty(t *testing.T) {
	rb := ringbuffer.New(5)

	// 从空缓冲区读取
	data := make([]byte, 1)
	n, err := rb.Read(data)
	if n != 0 || err == nil {
		t.Errorf("从空缓冲区读取应返回0和错误，实际返回%d和%v", n, err)
	}

	// 写入后读取全部数据
	rb.Write([]byte{1, 2})
	data = make([]byte, 2)
	rb.Read(data)

	// 再次从空缓冲区读取
	n, err = rb.Read(data)
	if n != 0 || err == nil {
		t.Errorf("读空后继续读取应返回0和错误，实际返回%d和%v", n, err)
	}
}

func TestSmallnest_ReadPartial(t *testing.T) {
	rb := ringbuffer.New(5)

	// 写入数据
	rb.Write([]byte{1, 2, 3, 4})

	// 请求读取超过可用数量的数据
	data := make([]byte, 6)
	n, err := rb.Read(data)
	if err != nil {
		t.Errorf("读取数据失败: %v", err)
	}
	if n != 4 {
		t.Errorf("请求读取6字节时应返回全部可用数据(4字节)，实际返回%d字节", n)
	}
	if !bytes.Equal(data[:n], []byte{1, 2, 3, 4}) {
		t.Errorf("读取的数据不匹配，期望[1,2,3,4]，实际为%v", data[:n])
	}
}

func TestSmallnest_ConcurrentAccess(t *testing.T) {
	rb := ringbuffer.New(100)
	done := make(chan bool)

	// 并发写入
	go func() {
		for i := 0; i < 100; i++ {
			rb.Write([]byte{byte(i)})
		}
		done <- true
	}()

	// 并发读取
	go func() {
		buf := make([]byte, 1)
		for i := 0; i < 100; i++ {
			rb.Read(buf)
		}
		done <- true
	}()

	// 等待两个goroutine完成
	<-done
	<-done
}
