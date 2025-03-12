package capture

import (
	"fmt"
	"math"
	"time"

	"github.com/gen2brain/malgo"
)

// Config 音频捕获配置
type Config struct {
	SampleRate       uint32
	Channels         uint32
	SilenceThreshold float64
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		SampleRate:       44100,
		Channels:         1,
		SilenceThreshold: 500,
	}
}

// AudioCapture 音频捕获器
type AudioCapture struct {
	config         *Config
	context        *malgo.AllocatedContext
	device         *malgo.Device
	processor      *AudioProcessor
	OnVolumeChange func(volume float64)
	OnError        func(err error)
}

// AudioProcessor 处理音频数据
type AudioProcessor struct {
	pcmData          []byte
	encodedData      []byte
	smoothedVolume   float64
	lastUpdate       time.Time
	lastSample       int16
	silenceThreshold float64
}

// NewAudioProcessor 创建新的音频处理器
func NewAudioProcessor(silenceThreshold float64) *AudioProcessor {
	return &AudioProcessor{
		lastUpdate:       time.Now(),
		silenceThreshold: silenceThreshold,
	}
}

// ProcessAudio 处理音频数据并返回音量
func (ap *AudioProcessor) ProcessAudio(samples []byte, frameCount uint32) float64 {
	// 保存原始PCM数据
	ap.pcmData = append(ap.pcmData, samples...)

	// 计算音量并压缩音频
	var sum float64
	for i := 0; i < len(samples); i += 2 {
		if i+1 >= len(samples) {
			break
		}
		// 将两个字节转换为16位整数
		sample := int16(samples[i]) | (int16(samples[i+1]) << 8)
		sum += math.Abs(float64(sample))

		// 简单的增量压缩：只记录与上一个样本的差值
		diff := sample - ap.lastSample
		if math.Abs(float64(sample)) < ap.silenceThreshold {
			// 静音处理：将低于阈值的样本记为0
			ap.encodedData = append(ap.encodedData, 0)
		} else if diff >= -127 && diff <= 127 {
			// 差值在一个字节范围内
			ap.encodedData = append(ap.encodedData, uint8(diff+128))
		} else {
			// 差值超出范围，记录完整样本
			ap.encodedData = append(ap.encodedData, 0xFF) // 标记为完整样本
			ap.encodedData = append(ap.encodedData, byte(sample&0xFF))
			ap.encodedData = append(ap.encodedData, byte(sample>>8))
		}
		ap.lastSample = sample
	}

	currentVolume := sum / float64(frameCount)
	ap.smoothedVolume = ap.smoothedVolume*0.7 + currentVolume*0.3

	return ap.smoothedVolume
}

// GetStats 获取音频统计信息
func (ap *AudioProcessor) GetStats() (pcmSize, encodedSize int, compressionRatio float64) {
	pcmSize = len(ap.pcmData)
	encodedSize = len(ap.encodedData)
	if pcmSize > 0 {
		compressionRatio = float64(encodedSize) / float64(pcmSize)
	}
	return
}

// GetPCMData 获取PCM数据
func (ap *AudioProcessor) GetPCMData() []byte {
	return ap.pcmData
}

// GetEncodedData 获取压缩后的数据
func (ap *AudioProcessor) GetEncodedData() []byte {
	return ap.encodedData
}

// NewAudioCapture 创建新的音频捕获器
func NewAudioCapture(config *Config) (*AudioCapture, error) {
	ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		return nil, fmt.Errorf("初始化上下文失败: %w", err)
	}

	capture := &AudioCapture{
		config:    config,
		context:   ctx,
		processor: NewAudioProcessor(config.SilenceThreshold),
	}

	return capture, nil
}

// Start 开始捕获音频
func (ac *AudioCapture) Start() error {
	deviceConfig := malgo.DefaultDeviceConfig(malgo.Capture)
	deviceConfig.Capture.Format = malgo.FormatS16
	deviceConfig.Capture.Channels = ac.config.Channels
	deviceConfig.SampleRate = ac.config.SampleRate
	deviceConfig.Alsa.NoMMap = 1

	onRecvFrames := func(pSample2, pSample []byte, framecount uint32) {
		volume := ac.processor.ProcessAudio(pSample, framecount)
		if ac.OnVolumeChange != nil {
			ac.OnVolumeChange(volume)
		}
	}

	device, err := malgo.InitDevice(ac.context.Context, deviceConfig, malgo.DeviceCallbacks{
		Data: onRecvFrames,
	})
	if err != nil {
		return fmt.Errorf("初始化设备失败: %w", err)
	}

	ac.device = device
	if err := device.Start(); err != nil {
		return fmt.Errorf("启动设备失败: %w", err)
	}

	return nil
}

// Stop 停止捕获音频
func (ac *AudioCapture) Stop() error {
	if ac.device != nil {
		ac.device.Uninit()
		ac.device = nil
	}
	return nil
}

// Close 关闭音频捕获器
func (ac *AudioCapture) Close() error {
	if err := ac.Stop(); err != nil {
		return err
	}

	if ac.context != nil {
		if err := ac.context.Uninit(); err != nil {
			return fmt.Errorf("关闭上下文失败: %w", err)
		}
		ac.context.Free()
		ac.context = nil
	}

	return nil
}

// GetProcessor 获取音频处理器
func (ac *AudioCapture) GetProcessor() *AudioProcessor {
	return ac.processor
}
