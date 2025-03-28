package recognition

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/joho/godotenv"
)

// 测试数据来自：
// ffmpeg -i <音频文件> -ar 16000 -ac 1 -f s16le -acodec pcm_s16le <输出文件>.pcm

func TestAliyunClient(t *testing.T) {
	// 加载.env文件
	err := godotenv.Load("../../.env")
	if err != nil {
		t.Fatalf("加载.env文件失败: %v", err)
	}

	// 从环境变量获取阿里云配置
	config := &AliyunConfig{
		AccessKeyID:     os.Getenv("ALIYUN_ACCESS_KEY_ID"),
		AccessKeySecret: os.Getenv("ALIYUN_ACCESS_KEY_SECRET"),
		AppKey:          os.Getenv("ALIYUN_APP_KEY"),
		Region:          os.Getenv("ALIYUN_REGION"),
	}

	// 检查配置是否完整
	if config.AccessKeyID == "" || config.AccessKeySecret == "" || config.AppKey == "" || config.Region == "" {
		t.Skip("缺少阿里云配置，跳过测试")
	}

	// 创建客户端
	client, err := NewAliyunClient(config, DefaultStartParam())
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}

	// 开始识别
	err = client.StartRecognition()
	if err != nil {
		t.Fatalf("启动识别失败: %v", err)
	}

	// 读取测试音频文件
	pcmData, err := ioutil.ReadFile("中国人.pcm")
	if err != nil {
		t.Fatalf("读取PCM文件失败: %v", err)
	}

	// 创建用于接收结果的通道
	done := make(chan struct{})
	var lastResult string
	var allResults []string

	// 在另一个 goroutine 中处理结果
	go func() {
		resultChan := client.GetResultChannel()
		completeChan := client.GetCompleteChannel()
		errorChan := client.GetErrorChannel()

		for {
			select {
			case result := <-resultChan:
				t.Logf("收到中间结果: %s", result)
				allResults = append(allResults, result)
			case result := <-completeChan:
				t.Logf("收到识别完成结果: %s", result)
				lastResult = result
				allResults = append(allResults, result)
			case err := <-errorChan:
				t.Errorf("识别错误: %v", err)
			case <-time.After(5 * time.Second):
				// 超时退出
				close(done)
				return
			}
		}
	}()

	// 分块发送音频数据
	chunkSize := 3200 // 每次发送200ms的音频数据
	for i := 0; i < len(pcmData); i += chunkSize {
		end := i + chunkSize
		if end > len(pcmData) {
			end = len(pcmData)
		}
		chunk := pcmData[i:end]

		err = client.SendAudioData(chunk)
		if err != nil {
			t.Fatalf("发送音频数据失败: %v", err)
		}

		// 模拟实时发送
		time.Sleep(100 * time.Millisecond)
	}

	// 停止识别
	err = client.StopRecognition()
	if err != nil {
		t.Fatalf("停止识别失败: %v", err)
	}

	// 等待结果处理完成
	<-done

	// 验证结果
	if lastResult == "" {
		t.Error("没有收到识别结果")
		return
	}

	// 验证最终结果
	expectedText := "我是一个中国人，我爱我的祖国。"
	if lastResult != expectedText {
		t.Errorf("最终识别结果不符合预期\n期望: %s\n实际: %s", expectedText, lastResult)
	}

	// 验证识别过程
	expectedPartialResults := []string{
		"我",
		"我是一",
		"我是一个中",
		"我是一个中国人",
		"我是一个中国人我爱",
		"我是一个中国人我爱我的",
		"我是一个中国人我爱我的祖国",
	}

	// 检查中间结果是否符合预期
	for i, expected := range expectedPartialResults {
		found := false
		for _, result := range allResults {
			if strings.HasPrefix(result, expected) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("未找到预期的中间结果[%d]: %s", i, expected)
		}
	}

	t.Logf("识别过程共产生 %d 个结果", len(allResults))
}

// TestTwoConsecutiveRecognitions 测试两次连续的语音识别周期
func TestTwoConsecutiveRecognitions(t *testing.T) {
	// 加载.env文件
	err := godotenv.Load("../../.env")
	if err != nil {
		t.Fatalf("加载.env文件失败: %v", err)
	}

	// 从环境变量获取阿里云配置
	config := &AliyunConfig{
		AccessKeyID:     os.Getenv("ALIYUN_ACCESS_KEY_ID"),
		AccessKeySecret: os.Getenv("ALIYUN_ACCESS_KEY_SECRET"),
		AppKey:          os.Getenv("ALIYUN_APP_KEY"),
		Region:          os.Getenv("ALIYUN_REGION"),
	}

	// 检查配置是否完整
	if config.AccessKeyID == "" || config.AccessKeySecret == "" || config.AppKey == "" || config.Region == "" {
		t.Skip("缺少阿里云配置，跳过测试")
	}

	// 创建客户端
	client, err := NewAliyunClient(config, DefaultStartParam())
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}

	// 读取测试音频文件
	pcmData, err := ioutil.ReadFile("中国人.pcm")
	if err != nil {
		t.Fatalf("读取PCM文件失败: %v", err)
	}

	// 执行第一次识别周期
	t.Log("开始第一次识别周期")
	result1 := performRecognitionCycle(t, client, pcmData)
	t.Logf("第一次识别结果: %s", result1)

	// 执行第二次识别周期
	// 读取测试音频文件
	pcmData, err = ioutil.ReadFile("帮我完成任务.pcm")
	if err != nil {
		t.Fatalf("读取PCM文件失败: %v", err)
	}
	t.Log("开始第二次识别周期")
	result2 := performRecognitionCycle(t, client, pcmData)
	t.Logf("第二次识别结果: %s", result2)

	// 验证两次识别都有结果
	if result1 == "" {
		t.Error("第一次识别没有收到结果")
	}
	if result2 == "" {
		t.Error("第二次识别没有收到结果")
	}

	t.Logf("两次识别周期完成，结果1: %s, 结果2: %s", result1, result2)
}

// performRecognitionCycle 执行一次完整的识别周期并返回结果
func performRecognitionCycle(t *testing.T, client *AliyunClient, audioData []byte) string {
	// 开始识别
	err := client.StartRecognition()
	if err != nil {
		t.Fatalf("启动识别失败: %v", err)
	}

	// 创建用于接收结果的通道
	done := make(chan struct{})
	var finalResult string

	// 在另一个 goroutine 中处理结果
	go func() {
		completeChan := client.GetCompleteChannel()
		errorChan := client.GetErrorChannel()

		for {
			select {
			case result := <-completeChan:
				t.Logf("收到识别完成结果: %s", result)
				finalResult = result
				close(done)
				return
			case err := <-errorChan:
				t.Logf("识别错误: %v", err)
				close(done)
				return
			case <-time.After(10 * time.Second):
				// 超时退出
				t.Log("识别超时")
				close(done)
				return
			}
		}
	}()

	// 分块发送音频数据
	chunkSize := 3200 // 每次发送200ms的音频数据
	for i := 0; i < len(audioData); i += chunkSize {
		end := i + chunkSize
		if end > len(audioData) {
			end = len(audioData)
		}
		chunk := audioData[i:end]

		err = client.SendAudioData(chunk)
		if err != nil {
			t.Fatalf("发送音频数据失败: %v", err)
		}

		// 模拟实时发送
		time.Sleep(100 * time.Millisecond)
	}

	// 停止识别
	err = client.StopRecognition()
	if err != nil {
		t.Fatalf("停止识别失败: %v", err)
	}

	// 等待结果处理完成
	<-done

	return finalResult
}
