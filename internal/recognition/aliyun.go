package recognition

import (
	"fmt"
	"sync"

	nls "github.com/aliyun/alibabacloud-nls-go-sdk"
)

// 阿里云Go SDK 文档：https://help.aliyun.com/zh/isi/developer-reference/sdk-for-go-1
// 生命周期：
// NewAliyunClient 创建阿里云语音识别客户端【无状态】
// StartRecognition 【需要无连接状态】开始语音识别WS连接【开始连接】
// SendAudioData 【需要连接状态】发送音频数据
// StopRecognition 【需要连接状态】停止语音识别，等待识别结果，然后【连接断开】
// ShutdownRecognition 【需要连接状态】关闭语音识别实例，不等待识别结果，立即【连接断开】

// StartRecognition->SendAudioData->StopRecognition 是一次识别的周期
//
// 注意：WS不可以长期连接，不识别时需要断开，它的最大空闲连接时间是60秒

type StartParam struct {
	// 5个SDK参数
	Format                         string // 音频格式:PCM、WAV、OPUS、SPEEX、AMR、MP3、AAC。
	SampleRate                     int    // 采样率:8000、16000 两种
	EnableIntermediateResult       bool   // 是否返回中间识别结果
	EnablePunctuationPrediction    bool   // 是否在后处理中添加标点
	EnableInverseTextNormalization bool   // 中文数字将转为阿拉伯数字输出
	// 4个自定义参数（API文档上的）
	DisableDisfluency    bool // disfluency 是否去除口语中的非正式表达(嗯嗯啊啊的语气词)
	EnableVoiceDetection bool // enable_voice_detection 是否开启语音检测
	MaxStartSilence      int  // max_start_silence 表示允许的最大开始静音时长
	MaxEndSilence        int  // max_end_silence 表示允许的最大结束静音时长
}

type AliyunClient struct {
	config        *AliyunConfig
	startParam    *StartParam
	resultChan    chan string
	completeChan  chan string
	errorChan     chan error
	stopChan      chan struct{}
	isRecognizing bool       // 正在识别中
	mutex         sync.Mutex // 识别切换锁

	sr     *nls.SpeechRecognition
	logger *nls.NlsLogger
}

// AliyunConfig 阿里云配置
type AliyunConfig struct {
	AccessKeyID     string
	AccessKeySecret string
	AppKey          string
	Region          string
}

// DefaultStartParam 默认的识别参数
func DefaultStartParam() *StartParam {
	return &StartParam{
		Format:                         "pcm",
		SampleRate:                     16000,
		EnableIntermediateResult:       true,
		EnablePunctuationPrediction:    true,
		EnableInverseTextNormalization: true,
		DisableDisfluency:              true,
		EnableVoiceDetection:           true,
		MaxStartSilence:                5000,
		MaxEndSilence:                  3000,
	}
}

// NewAliyunClient 创建新的阿里云语音识别客户端
func NewAliyunClient(cfg *AliyunConfig, startParam *StartParam) (*AliyunClient, error) {
	ac := &AliyunClient{
		config:        cfg,
		startParam:    startParam,
		resultChan:    make(chan string, 10),
		completeChan:  make(chan string, 10),
		errorChan:     make(chan error, 10),
		stopChan:      make(chan struct{}),
		isRecognizing: false,
		logger:        nls.DefaultNlsLog(),
	}
	ac.logger.SetLogSil(true)

	// 创建阿里云NLS客户端配置
	wsUrl := fmt.Sprintf("wss://nls-gateway-%s.aliyuncs.com/ws/v1", ac.config.Region)
	config, err := nls.NewConnectionConfigWithAKInfoDefault(
		wsUrl,
		ac.config.AppKey,
		ac.config.AccessKeyID,
		ac.config.AccessKeySecret,
	)
	if err != nil {
		return nil, fmt.Errorf("创建连接配置失败: %v", err)
	}

	ac.sr, err = nls.NewSpeechRecognition(config, ac.logger,
		ac.onTaskFailed, ac.onStarted, ac.onResultChanged,
		ac.onCompleted, ac.onClose, ac.logger)
	if err != nil {
		return nil, fmt.Errorf("创建语音识别实例失败: %v", err)
	}

	return ac, nil
}

// StartRecognition 开始语音识别
func (ac *AliyunClient) StartRecognition() error {
	ac.mutex.Lock()
	defer ac.mutex.Unlock()

	if ac.isRecognizing {
		return fmt.Errorf("StartRecognition 重复启动")
	}
	nlsStartParam := nls.SpeechRecognitionStartParam{
		Format:                         ac.startParam.Format,
		SampleRate:                     ac.startParam.SampleRate,
		EnableIntermediateResult:       ac.startParam.EnableIntermediateResult,
		EnablePunctuationPrediction:    ac.startParam.EnablePunctuationPrediction,
		EnableInverseTextNormalization: ac.startParam.EnableInverseTextNormalization,
	}

	// 启动识别
	ready, err := ac.sr.Start(nlsStartParam, map[string]interface{}{
		"disfluency":             ac.startParam.DisableDisfluency,
		"enable_voice_detection": ac.startParam.EnableVoiceDetection,
		"max_start_silence":      ac.startParam.MaxStartSilence,
		"max_end_silence":        ac.startParam.MaxEndSilence,
	})

	if err != nil {
		return fmt.Errorf("StartRecognition Start失败: %v", err)
	}

	// 是否完成连接就看这个ready
	ac.isRecognizing = <-ready
	if !ac.isRecognizing {
		return fmt.Errorf("StartRecognition WS连接失败")
	}
	return nil
}

// SendAudioData 发送音频数据
func (ac *AliyunClient) SendAudioData(data []byte) error {
	ac.mutex.Lock()
	defer ac.mutex.Unlock()

	if !ac.isRecognizing {
		return fmt.Errorf("语音识别未连接")
	}

	return ac.sr.SendAudioData(data)
}

// StopRecognition 停止语音识别任务
func (ac *AliyunClient) StopRecognition() error {
	ac.mutex.Lock()
	defer ac.mutex.Unlock()

	if !ac.isRecognizing {
		return nil
	}

	// 停止识别并等待结果
	ready, err := ac.sr.Stop()
	if err != nil {
		return fmt.Errorf("停止语音识别失败: %v", err)
	}

	// 这里是等待识别完成事件或任务失败事件，我们不期望在这里得到这个结果。
	<-ready
	// 停止并关闭连接
	ac.sr.Shutdown()
	ac.isRecognizing = false
	return nil
}

// Close 关闭连接
func (ac *AliyunClient) ShutdownRecognition() {
	ac.mutex.Lock()
	defer ac.mutex.Unlock()

	ac.sr.Shutdown()

	ac.isRecognizing = false

}
