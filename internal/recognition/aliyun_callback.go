package recognition

import (
	"encoding/json"
	"fmt"
	"log"
)

// status错误码：

// 41040201 请保持实时速率发送，发送完成后及时关闭链接。
// 41010101 当前实时语音识别只支持8000 Hz和16000 Hz两种采样率格式的音频。
// 40000004 长时间没有发送任何数据，超过10s后服务端会返回此错误信息。
// 40270002 从音频中没有识别出有效文本。
// 41010104 发送的语音时长超过60s限制
// 41010105 纯静音数据或噪音数据，导致无法检测出任何有效语音。

// Name结果数据名称
// RecognitionCompleted 识别完成
// RecognitionResultChanged 表示获取到中间识别结果
// 解析识别结果
type recognitionResult struct {
	Header struct {
		Namespace  string `json:"namespace"`
		Name       string `json:"name"`
		Status     int    `json:"status"` // 20000000 表示识别成功
		MessageId  string `json:"message_id"`
		TaskId     string `json:"task_id"`
		StatusText string `json:"status_text"`
	} `json:"header"`
	Payload struct {
		Result string `json:"result"`
	} `json:"payload"`
}

// extractText 从JSON响应中提取文本
func extractText(jsonStr string) (*recognitionResult, error) {
	var result recognitionResult
	err := json.Unmarshal([]byte(jsonStr), &result)
	if err != nil {
		return nil, fmt.Errorf("解析JSON失败: %v", err)
	}
	return &result, nil
}

// onTaskFailed 处理识别任务失败的回调
func (ac *AliyunClient) onTaskFailed(text string, param interface{}) {
	result, err := extractText(text)
	if err != nil {
		ac.errorChan <- err
		return
	}
	// 任务失败如果是 status==41010105 && status_text=="SILENT_SPEECH"，说明是开始后但是超过max_start_silence没有识别到声音
	// 如果是这样，应该发送到完成Chan而不是err
	if result.Header.Status == 41010105 && result.Header.StatusText == "SILENT_SPEECH" {
		// 输出调试警告信息
		log.Printf("开始识别后 %d ms 未识别到声音，结束识别", ac.startParam.MaxStartSilence)
		ac.completeChan <- ""
		return
	}
	ac.errorChan <- fmt.Errorf("识别失败: %s", text)
}

func (ac *AliyunClient) onStarted(text string, param interface{}) {
	// 这些回调应该都是基于WS消息的，不是WS连接状态级别的东西
	log.Printf("onStarted: %s", text)
}

// onResultChanged 中间结果
func (ac *AliyunClient) onResultChanged(text string, param interface{}) {
	result, err := extractText(text)
	if err != nil {
		ac.errorChan <- err
		return
	}
	ac.resultChan <- result.Payload.Result
}

// onCompleted 处理识别完成
func (ac *AliyunClient) onCompleted(text string, param interface{}) {
	result, err := extractText(text)
	if err != nil {
		ac.errorChan <- err
		return
	}
	ac.completeChan <- result.Payload.Result
}

func (ac *AliyunClient) onClose(param interface{}) {
	// 这些回调应该都是基于WS消息的，不是WS连接状态级别的东西
}

// GetResultChannel 获取结果通道
func (ac *AliyunClient) GetResultChannel() <-chan string {
	return ac.resultChan
}

// GetCompleteChannel 获取完成通道
func (ac *AliyunClient) GetCompleteChannel() <-chan string {
	return ac.completeChan
}

// GetErrorChannel 获取错误通道
func (ac *AliyunClient) GetErrorChannel() <-chan error {
	return ac.errorChan
}
