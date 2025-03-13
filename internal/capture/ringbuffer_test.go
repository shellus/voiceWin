package capture

import (
	"bytes"
	"testing"
)

func TestNewAudioCapture(t *testing.T) {
	// 测试创建新的音频捕获实例
	ac := NewAudioCapture()
	if ac == nil {
		t.Error("NewAudioCapture返回了nil")
	}

	// 测试默认配置
	if ac.config.SampleRate != 16000 {
		t.Errorf("期望采样率为16000，实际为%d", ac.config.SampleRate)
	}
	if ac.config.Channels != 1 {
		t.Errorf("期望通道数为1，实际为%d", ac.config.Channels)
	}

	// 测试处理器初始化
	if ac.processor == nil {
		t.Error("音频处理器未初始化")
	}
	if ac.processor.ringBuffer == nil {
		t.Error("环形缓冲区未初始化")
	}

	// 测试回调函数初始化为nil
	if ac.OnVolumeChange != nil {
		t.Error("OnVolumeChange应初始化为nil")
	}
	if ac.OnAudioData != nil {
		t.Error("OnAudioData应初始化为nil")
	}
	if ac.OnError != nil {
		t.Error("OnError应初始化为nil")
	}
}

func TestAudioCapture_Close(t *testing.T) {
	ac := NewAudioCapture()
	if ac == nil {
		t.Fatal("创建音频捕获实例失败")
	}

	// 测试关闭未启动的实例
	err := ac.Close()
	if err != nil {
		t.Errorf("关闭未启动的实例应返回nil，实际返回%v", err)
	}

	// 测试关闭已启动的实例
	err = ac.Start()
	if err != nil {
		t.Fatalf("启动音频捕获失败: %v", err)
	}

	err = ac.Close()
	if err != nil {
		t.Errorf("关闭已启动的实例失败: %v", err)
	}

	// 测试重复关闭
	err = ac.Close()
	if err != nil {
		t.Errorf("重复关闭应返回nil，实际返回%v", err)
	}
}

func TestRingBuffer_Overflow(t *testing.T) {
	rb := NewRingBuffer(5)

	// 测试1：写入超过缓冲区大小的数据
	data := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	n := rb.Write(data)
	if n != 5 {
		t.Errorf("期望写入5字节（缓冲区大小），实际写入%d字节", n)
	}

	// 验证只保留了最后5个字节
	readData := rb.Read(5)
	if !bytes.Equal(readData, []byte{4, 5, 6, 7, 8}) {
		t.Errorf("读取的数据不匹配，期望[4,5,6,7,8]，实际为%v", readData)
	}

	// 测试2：缓冲区满时继续写入
	rb.Write([]byte{1, 2, 3, 4, 5}) // 先填满缓冲区
	n = rb.Write([]byte{6, 7})      // 再写入新数据
	if n != 2 {
		t.Errorf("期望写入2字节，实际写入%d字节", n)
	}

	// 验证自动腾出空间并保留最新数据
	readData = rb.Read(5)
	if !bytes.Equal(readData, []byte{3, 4, 5, 6, 7}) {
		t.Errorf("读取的数据不匹配，期望[3,4,5,6,7]，实际为%v", readData)
	}

	// 测试3：连续写入小块数据直到溢出
	rb = NewRingBuffer(3)
	rb.Write([]byte{1})
	rb.Write([]byte{2})
	rb.Write([]byte{3})
	rb.Write([]byte{4}) // 这次写入应该会自动腾出空间

	readData = rb.Read(3)
	if !bytes.Equal(readData, []byte{2, 3, 4}) {
		t.Errorf("读取的数据不匹配，期望[2,3,4]，实际为%v", readData)
	}

	// 测试4：写入空数据
	n = rb.Write([]byte{})
	if n != 0 {
		t.Errorf("写入空数据应返回0，实际返回%d", n)
	}

	// 测试5：写入nil
	n = rb.Write(nil)
	if n != 0 {
		t.Errorf("写入nil应返回0，实际返回%d", n)
	}
}
