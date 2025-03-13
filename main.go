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

	recogCfg := &recognition.RecognitionConfig{
		Format:            "pcm",
		SampleRate:        16000,
		EnablePunctuation: true,
		EnableITN:         true,
	}

	// 初始化阿里云客户端
	aliyunClient := recognition.NewAliyunClient(aliyunCfg, recogCfg)

	// 连接到阿里云服务
	if err := aliyunClient.Connect(); err != nil {
		log.Fatalf("连接到阿里云服务失败: %v", err)
	}
	defer aliyunClient.Close()

	// 启动语音识别
	if err := aliyunClient.StartRecognition(); err != nil {
		log.Fatalf("启动语音识别失败: %v", err)
	}

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

	// 启动音频捕获
	if err := audioCapture.Start(); err != nil {
		log.Fatalf("启动音频捕获失败: %v", err)
	}

	fmt.Println("开始录音...按 Ctrl+C 停止")

	// 创建通道用于接收识别结果
	resultChan := aliyunClient.GetResultChannel()
	errorChan := aliyunClient.GetErrorChannel()

	// 启动识别结果处理协程
	go func() {
		for {
			select {
			case result := <-resultChan:
				fmt.Printf("\n识别结果: %s\n", result)
			case err := <-errorChan:
				log.Printf("\n错误: %v\n", err)
			}
		}
	}()

	// 等待中断信号
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan

	fmt.Println("\n正在关闭程序...")

	// 停止语音识别
	if err := aliyunClient.StopRecognition(); err != nil {
		log.Printf("停止语音识别失败: %v", err)
	}
}
