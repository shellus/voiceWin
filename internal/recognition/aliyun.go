package recognition

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"

	nls "github.com/aliyun/alibabacloud-nls-go-sdk"
)

// AliyunClient 阿里云语音识别客户端
type AliyunClient struct {
	config        *AliyunConfig
	recogConfig   *RecognitionConfig
	resultChan    chan string
	errorChan     chan error
	stopChan      chan struct{}
	isConnected   bool
	isRecognizing bool
	taskID        string
	recognizer    *nls.SpeechRecognition
	logger        *nls.NlsLogger
	mutex         sync.Mutex
}

// AliyunConfig 阿里云配置
type AliyunConfig struct {
	AccessKeyID     string
	AccessKeySecret string
	AppKey          string
	Region          string
}

// RecognitionConfig 识别配置
type RecognitionConfig struct {
	Format            string
	SampleRate        int
	EnablePunctuation bool
	EnableITN         bool
}

// 解析识别结果
type recognitionResult struct {
	Header struct {
		Status     int    `json:"status"`
		StatusText string `json:"status_text"`
	} `json:"header"`
	Payload struct {
		Result string `json:"result"`
	} `json:"payload"`
}

// extractText 从JSON响应中提取文本
func extractText(jsonStr string) (string, error) {
	var result recognitionResult
	err := json.Unmarshal([]byte(jsonStr), &result)
	if err != nil {
		return "", fmt.Errorf("解析JSON失败: %v", err)
	}

	if result.Header.Status != 20000000 {
		return "", fmt.Errorf("识别失败: %s", result.Header.StatusText)
	}

	return result.Payload.Result, nil
}

// onTaskFailed 处理识别任务失败的回调
func (ac *AliyunClient) onTaskFailed(text string, param interface{}) {
	ac.errorChan <- fmt.Errorf("识别失败: %s", text)
}

// onStarted 处理识别开始的回调
func (ac *AliyunClient) onStarted(text string, param interface{}) {
	log.Printf("识别开始: %s", text)
}

// onResultChanged 处理识别结果变化的回调
func (ac *AliyunClient) onResultChanged(text string, param interface{}) {
	result, err := extractText(text)
	if err != nil {
		ac.errorChan <- err
		return
	}
	ac.resultChan <- result
}

// onCompleted 处理识别完成的回调
func (ac *AliyunClient) onCompleted(text string, param interface{}) {
	result, err := extractText(text)
	if err != nil {
		ac.errorChan <- err
		return
	}
	ac.resultChan <- result
}

// onClose 处理连接关闭的回调
func (ac *AliyunClient) onClose(param interface{}) {
	log.Println("连接已关闭")
	ac.isConnected = false
}

// NewAliyunClient 创建新的阿里云语音识别客户端
func NewAliyunClient(cfg *AliyunConfig, recogCfg *RecognitionConfig) *AliyunClient {
	return &AliyunClient{
		config:        cfg,
		recogConfig:   recogCfg,
		resultChan:    make(chan string, 10),
		errorChan:     make(chan error, 10),
		stopChan:      make(chan struct{}),
		isConnected:   false,
		isRecognizing: false,
		logger:        nls.DefaultNlsLog(),
	}
}

// Connect 连接到阿里云语音识别服务
func (ac *AliyunClient) Connect() error {
	ac.mutex.Lock()
	defer ac.mutex.Unlock()

	if ac.isConnected {
		return nil
	}

	// 创建阿里云NLS客户端配置
	wsUrl := fmt.Sprintf("wss://nls-gateway-%s.aliyuncs.com/ws/v1", ac.config.Region)
	config, err := nls.NewConnectionConfigWithAKInfoDefault(
		wsUrl,
		ac.config.AppKey,
		ac.config.AccessKeyID,
		ac.config.AccessKeySecret,
	)
	if err != nil {
		return fmt.Errorf("创建配置失败: %v", err)
	}

	// 创建语音识别实例
	sr, err := nls.NewSpeechRecognition(config, ac.logger,
		ac.onTaskFailed, ac.onStarted, ac.onResultChanged,
		ac.onCompleted, ac.onClose, ac.logger)
	if err != nil {
		return fmt.Errorf("创建语音识别器失败: %v", err)
	}

	ac.recognizer = sr
	ac.isConnected = true
	return nil
}

// StartRecognition 开始语音识别
func (ac *AliyunClient) StartRecognition() error {
	ac.mutex.Lock()
	defer ac.mutex.Unlock()

	if !ac.isConnected {
		return fmt.Errorf("未连接到阿里云语音识别服务")
	}

	if ac.isRecognizing {
		return nil
	}

	// 设置识别参数
	param := nls.DefaultSpeechRecognitionParam()
	param.Format = ac.recogConfig.Format
	param.SampleRate = ac.recogConfig.SampleRate
	param.EnableInverseTextNormalization = ac.recogConfig.EnableITN

	// 启动识别
	ready, err := ac.recognizer.Start(param, nil)
	if err != nil {
		return fmt.Errorf("启动语音识别失败: %v", err)
	}

	// 等待启动完成
	select {
	case done := <-ready:
		if !done {
			return fmt.Errorf("启动失败")
		}
	}

	ac.isRecognizing = true
	return nil
}

// SendAudioData 发送音频数据
func (ac *AliyunClient) SendAudioData(data []byte) error {
	if !ac.isRecognizing {
		return fmt.Errorf("语音识别未启动")
	}

	return ac.recognizer.SendAudioData(data)
}

// StopRecognition 停止语音识别
func (ac *AliyunClient) StopRecognition() error {
	ac.mutex.Lock()
	defer ac.mutex.Unlock()

	if !ac.isRecognizing {
		return nil
	}

	// 停止识别并等待结果
	ready, err := ac.recognizer.Stop()
	if err != nil {
		return fmt.Errorf("停止语音识别失败: %v", err)
	}

	// 等待停止完成
	select {
	case done := <-ready:
		if !done {
			return fmt.Errorf("停止失败")
		}
	}

	ac.isRecognizing = false
	return nil
}

// Close 关闭连接
func (ac *AliyunClient) Close() error {
	ac.mutex.Lock()
	defer ac.mutex.Unlock()

	if !ac.isConnected {
		return nil
	}

	if ac.recognizer != nil {
		ac.recognizer.Shutdown()
	}

	ac.isConnected = false
	return nil
}

// GetResultChannel 获取结果通道
func (ac *AliyunClient) GetResultChannel() <-chan string {
	return ac.resultChan
}

// GetErrorChannel 获取错误通道
func (ac *AliyunClient) GetErrorChannel() <-chan error {
	return ac.errorChan
}
