package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/shellus/voiceWin/internal/recognition"
)

var (
	isDebugging bool
)

func init() {
	// 设置命令行参数
	flag.BoolVar(&isDebugging, "debug", false, "启用调试模式")
	flag.BoolVar(&isDebugging, "d", false, "启用调试模式（简写）")

	// 解析命令行参数
	flag.Parse()

	// 加载 .env 文件
	if err := godotenv.Load(); err != nil {
		log.Printf("警告: 未能加载 .env 文件: %v", err)
	}
}

// 主程序逻辑
func startVoiceWin() {
	fmt.Println("启动 voiceWin...")
	fmt.Println("按 Ctrl+C 退出程序")

	// 创建简单的配置
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

	// 创建键盘输入器
	keyboardInput := input.NewKeyboardInput()

	// 创建通道用于接收识别结果
	resultChan := aliyunClient.GetResultChannel()
	errorChan := aliyunClient.GetErrorChannel()

	// 启动识别结果处理协程
	go func() {
		for {
			select {
			case result := <-resultChan:
				// 将识别结果输入到当前活动窗口
				fmt.Printf("识别结果: %s\n", result)
				keyboardInput.TypeText(result)
			case err := <-errorChan:
				log.Printf("错误: %v", err)
			}
		}
	}()

	// 模拟按下快捷键启动识别
	fmt.Println("模拟按下Alt+V启动识别...")
	startRecognition(aliyunClient)

	// 等待退出信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\n正在关闭 voiceWin...")
	stopRecognition(aliyunClient)
}

// 启动语音识别
func startRecognition(aliyunClient *recognition.AliyunClient) {
	// 启动语音识别
	if err := aliyunClient.StartRecognition(); err != nil {
		log.Printf("启动语音识别失败: %v", err)
		return
	}

	fmt.Println("语音识别已启动，请开始说话...")
}

// 停止语音识别
func stopRecognition(aliyunClient *recognition.AliyunClient) {
	// 停止语音识别
	if err := aliyunClient.StopRecognition(); err != nil {
		log.Printf("停止语音识别失败: %v", err)
	}

	fmt.Println("语音识别已停止")
}

func main() {
	startVoiceWin()
}
