package capture

import (
	"math"
	"time"
)
// Config 音频捕获配置
type Config struct {
	SampleRate       uint32
	Channels         uint32
	SilenceThreshold float64
	BufferDuration   time.Duration // 音频缓冲区时长
	CallbackInterval time.Duration // 数据回调间隔
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		SampleRate:       44100,
		Channels:         1,
		SilenceThreshold: 500,
		BufferDuration:   time.Second,           // 默认1秒缓冲
		CallbackInterval: 20 * time.Millisecond, // 默认20ms回调一次
	}
}

// AudioProcessor 处理音频数据
type AudioProcessor struct {
	config           *Config
	ringBuffer       *RingBuffer // 环形缓冲区
	smoothedVolume   float64
	silenceThreshold float64
	bufferSize       int
}

// NewAudioProcessor 创建新的音频处理器
func NewAudioProcessor(config *Config) *AudioProcessor {
	// 计算缓冲区大小：采样率 * 通道数 * 采样大小(字节) * 缓冲时长(秒)
	bufferSize := int(config.SampleRate * config.Channels * 2 * uint32(config.BufferDuration.Seconds()))

	return &AudioProcessor{
		config:           config,
		ringBuffer:       NewRingBuffer(bufferSize),
		silenceThreshold: config.SilenceThreshold,
		bufferSize:       bufferSize,
	}
}

// ProcessAudio 处理音频数据并返回音量
func (ap *AudioProcessor) ProcessAudio(samples []byte, frameCount uint32) float64 {
	// 写入环形缓冲区
	ap.ringBuffer.Write(samples)

	// 计算音量
	var sum float64
	for i := 0; i < len(samples); i += 2 {
		if i+1 >= len(samples) {
			break
		}
		sample := int16(samples[i]) | (int16(samples[i+1]) << 8)
		sum += math.Abs(float64(sample))
	}

	currentVolume := sum / float64(frameCount)
	ap.smoothedVolume = ap.smoothedVolume*0.7 + currentVolume*0.3

	return ap.smoothedVolume
}

// GetPCMData 获取PCM数据
func (ap *AudioProcessor) GetPCMData() []byte {
	return ap.ringBuffer.Read(ap.bufferSize)
}

// GetStats 获取音频统计信息
func (ap *AudioProcessor) GetStats() (pcmSize, encodedSize int, compressionRatio float64) {
	pcmSize = ap.ringBuffer.Size()
	encodedSize = 0
	compressionRatio = 0
	return
}
