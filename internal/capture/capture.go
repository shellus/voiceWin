package capture

import (
	"fmt"
	"math"
	"time"

	"github.com/gen2brain/malgo"
)

// AudioCapture 音频捕获器
type AudioCapture struct {
	config         *Config
	context        *malgo.AllocatedContext
	device         *malgo.Device
	processor      *AudioProcessor
	OnVolumeChange func(volume float64)
	OnAudioData    func()
	OnError        func(err error)
	lastDataCall   time.Time // 上次数据回调的时间
	lastVolume     float64   // 上次音量值
}

// NewAudioCapture 创建新的音频捕获器
func NewAudioCapture() *AudioCapture {
	config := DefaultConfig()
	config.SampleRate = 16000 // 设置采样率为16kHz

	context, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		return nil
	}

	return &AudioCapture{
		config:    config,
		context:   context,
		processor: NewAudioProcessor(config),
	}
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

		// 只在音量变化时触发回调
		if ac.OnVolumeChange != nil && math.Abs(volume-ac.lastVolume) > 1.0 {
			ac.OnVolumeChange(volume)
			ac.lastVolume = volume
		}

		// 节流处理数据回调
		if ac.OnAudioData != nil {
			now := time.Now()
			if now.Sub(ac.lastDataCall) >= ac.config.CallbackInterval {
				ac.OnAudioData()
				ac.lastDataCall = now
			}
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

func (ac *AudioCapture) GetPCMData() []byte {
	return ac.processor.GetPCMData()
}
