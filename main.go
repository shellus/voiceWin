package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/joho/godotenv"
	"github.com/shellus/voiceWin/internal/capture"
	"github.com/shellus/voiceWin/internal/recognition"
)

var stopChan = make(chan os.Signal, 1)
var doneChan = make(chan struct{})

func chanWait(completeChan <-chan string, errorChan <-chan error) {
	for {
		select {
		case result := <-completeChan:
			onResult(result)
			return
		case err := <-errorChan:
			onError(err)
			return
		}
	}
}

func onResult(result string) {
	fmt.Printf("\n识别结果: %s\n", result)
	close(doneChan)
}

func onError(err error) {
	log.Printf("\n错误: %v\n", err)
	close(doneChan)
}

func main() {
	// 加载环境变量
	if err := godotenv.Load(); err != nil {
		log.Printf("警告: 未能加载 .env 文件: %v", err)
	}

	// 创建音频捕获器
	audioCapture := capture.NewAudioCapture()

	defer audioCapture.Close()

	// 创建阿里云配置
	aliyunCfg := &recognition.AliyunConfig{
		AccessKeyID:     os.Getenv("ALIYUN_ACCESS_KEY_ID"),
		AccessKeySecret: os.Getenv("ALIYUN_ACCESS_KEY_SECRET"),
		AppKey:          os.Getenv("ALIYUN_APP_KEY"),
		Region:          os.Getenv("ALIYUN_REGION"),
	}

	// 1. 初始化阿里云客户端
	aliyunClient, err := recognition.NewAliyunClient(aliyunCfg, recognition.DefaultStartParam())
	if err != nil {
		log.Fatalf("初始化阿里云客户端失败: %v", err)
	}

	// 2. 启动语音识别
	if err := aliyunClient.StartRecognition(); err != nil {
		log.Fatalf("启动语音识别失败: %v", err)
	}

	// 3. 启动音频捕获
	audioCapture.OnVolumeChange = func(volume float64) {
		fmt.Printf("\r音量: %f", volume)
	}
	audioCapture.OnAudioData = func() {
		pcmData := audioCapture.GetPCMData()
		if len(pcmData) == 0 {
			// 当audioCapture.Start()后，采集器触发了回调，应该20ms收到一次采集数据的
			log.Fatalf("音频数据为空")
		}
		if err := aliyunClient.SendAudioData(pcmData); err != nil {
			log.Printf("发送音频数据失败: %v", err)
		}
	}
	if err := audioCapture.Start(); err != nil {
		log.Fatalf("启动音频捕获失败: %v", err)
	}

	fmt.Println("开始录音...按 Ctrl+C 停止")

	go chanWait(aliyunClient.GetCompleteChannel(), aliyunClient.GetErrorChannel())

	// 注意，退出分为3种情况：
	// 1. 识别完成：触发onResult，关闭doneChan，执行ShutdownRecognition
	// 2. 识别失败：触发onError，关闭doneChan，执行ShutdownRecognition
	// 3. Ctrl+C：执行StopRecognition，等待doneChan，然后执行ShutdownRecognition

	signal.Notify(stopChan, os.Interrupt)
	select {
	case <-doneChan:
		// 识别完成或失败，直接关闭
		fmt.Println("\n正在关闭...")
		aliyunClient.ShutdownRecognition()
		audioCapture.Close()
	case <-stopChan:
		fmt.Println("\n正在停止识别...")
		// 先停止音频捕获
		if err := audioCapture.Stop(); err != nil {
			log.Printf("停止音频捕获失败: %v", err)
		}
		// 停止识别并等待完成
		if err := aliyunClient.StopRecognition(); err != nil {
			log.Printf("停止识别失败: %v", err)
		}
		// 等待识别完成或失败
		// 因为select已经进入了case <-stopChan，所以我们这里需要手动等待doneChan
		<-doneChan
		fmt.Println("\n正在关闭...")
		// 完全关闭
		aliyunClient.ShutdownRecognition()
		audioCapture.Close()
	}
}
