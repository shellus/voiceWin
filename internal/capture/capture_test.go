package capture

import (
	"testing"
	"time"
)

func TestAudioProcessor(t *testing.T) {
	// 创建处理器
	processor := NewAudioProcessor(500)

	// 创建模拟的音频数据（16位PCM）
	// 生成一个简单的正弦波
	sampleData := make([]byte, 1000)
	for i := 0; i < len(sampleData); i += 2 {
		// 模拟一个16位的样本值
		sample := int16(10000) // 固定振幅
		sampleData[i] = byte(sample & 0xFF)
		sampleData[i+1] = byte(sample >> 8)
	}

	// 测试音频处理
	volume := processor.ProcessAudio(sampleData, uint32(len(sampleData)/2))
	if volume <= 0 {
		t.Errorf("期望音量大于0，实际获得 %f", volume)
	}

	// 测试统计信息
	pcmSize, encodedSize, ratio := processor.GetStats()
	if pcmSize != len(sampleData) {
		t.Errorf("PCM大小不匹配，期望 %d，实际 %d", len(sampleData), pcmSize)
	}
	if encodedSize <= 0 {
		t.Errorf("压缩数据大小应该大于0，实际 %d", encodedSize)
	}
	if ratio <= 0 || ratio > 1 {
		t.Errorf("压缩比应该在0到1之间，实际 %f", ratio)
	}

	// 测试获取数据
	pcmData := processor.GetPCMData()
	if len(pcmData) != len(sampleData) {
		t.Errorf("PCM数据长度不匹配，期望 %d，实际 %d", len(sampleData), len(pcmData))
	}

	encodedData := processor.GetEncodedData()
	if len(encodedData) != encodedSize {
		t.Errorf("压缩数据长度不匹配，期望 %d，实际 %d", encodedSize, len(encodedData))
	}
}

func TestAudioCapture(t *testing.T) {
	// 创建配置
	config := DefaultConfig()
	if config.SampleRate != 44100 {
		t.Errorf("默认采样率应为44100，实际为 %d", config.SampleRate)
	}

	// 创建捕获器
	capture, err := NewAudioCapture(config)
	if err != nil {
		t.Fatalf("创建音频捕获器失败: %v", err)
	}
	defer capture.Close()

	// 测试音量变化回调
	volumeReceived := make(chan float64, 1)
	capture.OnVolumeChange = func(volume float64) {
		select {
		case volumeReceived <- volume:
		default:
		}
	}

	// 启动捕获
	if err := capture.Start(); err != nil {
		t.Fatalf("启动音频捕获失败: %v", err)
	}

	// 等待一小段时间以接收一些音频数据
	select {
	case volume := <-volumeReceived:
		t.Logf("接收到音量: %f", volume)
	case <-time.After(time.Second):
		t.Error("超时：未收到音量数据")
	}

	// 停止捕获
	if err := capture.Stop(); err != nil {
		t.Errorf("停止音频捕获失败: %v", err)
	}

	// 获取处理器并检查统计信息
	processor := capture.GetProcessor()
	pcmSize, encodedSize, ratio := processor.GetStats()
	t.Logf("PCM大小: %d, 压缩大小: %d, 压缩比: %f", pcmSize, encodedSize, ratio)
}
