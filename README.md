# voiceWin

voiceWin 是一个基于阿里云语音识别服务的 Windows 语音转文字工具。

## 项目目标

开发一个轻量级的语音识别工具，通过阿里云语音识别服务，实现实时语音转文字功能。

## 当前功能

- ✅ 麦克风音频采集
- ✅ 阿里云语音识别接口对接
- ✅ 命令行单次启动识别
- ⚪ 命令行多次识别
- ⚪ 发送到当前光标输入框
- ⚪ 全局键盘监听（可选）
- ⚪ 分贝开始并附带前1秒缓冲区，使用阿里云静音结束检测（全自动无需按键开始结束）
- ⚪ opus（ogg）编码
- ⚪ UI：WEB或其他界面

## 技术特点

- 使用 Go 语言开发
- 集成阿里云实时语音识别服务
- 支持实时音量显示
- 支持 PCM 音频数据采集
- 优雅的程序退出处理

## 使用前提

1. 需要阿里云账号
2. 需要开通阿里云实时语音识别服务
3. 为子账号授予AliyunNLSSpeechServiceAccess权限
3. 配置以下环境变量：
   - ALIYUN_ACCESS_KEY_ID
   - ALIYUN_ACCESS_KEY_SECRET
   - ALIYUN_APP_KEY
   - ALIYUN_REGION

文档：https://help.aliyun.com/zh/isi/product-overview/billing-10?spm=a2c4g.11186623.0.0.563068354s54pf
> 价格：试用3个月免费，然后3.5元/千次

## 开发计划

1. 实现多次连续识别功能
2. 添加文本输入功能
3. 开发用户界面
4. 优化音频编码方式
5. 添加智能语音激活功能

## 项目状态

目前项目处于早期开发阶段，已完成基础的音频采集和语音识别功能。后续将继续开发更多功能，欢迎贡献代码或提出建议。 

## CGO依赖
malgo 需要CGO编译，如果没有gcc请使用scoop安装gcc，然后再安装或修复go，否则会报错
```shell
$ go build -o voiceWin.exe main.go
# github.com/shellus/voiceWin/internal/capture
internal\capture\capture.go:14:24: undefined: malgo.AllocatedContext
internal\capture\capture.go:15:24: undefined: malgo.Device
internal\capture\capture.go:29:24: undefined: malgo.InitContext
internal\capture\capture.go:29:47: undefined: malgo.ContextConfig
internal\capture\capture.go:43:24: undefined: malgo.DefaultDeviceConfig
internal\capture\capture.go:68:23: undefined: malgo.InitDevice
internal\capture\capture.go:68:74: undefined: malgo.DeviceCallbacks

```