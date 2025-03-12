package hotkey

import (
	"log"
)

// KeyboardInput 表示键盘输入器
type KeyboardInput struct {
	// 可以在这里添加配置参数
}

// NewKeyboardInput 创建新的键盘输入器
func NewKeyboardInput() *KeyboardInput {
	return &KeyboardInput{}
}

// TypeText 将文本输入到当前活动窗口
func (ki *KeyboardInput) TypeText(text string) error {
	// 模拟键盘输入
	log.Printf("模拟键盘输入: %s", text)
	return nil
}

// PressKey 模拟按下指定键
func (ki *KeyboardInput) PressKey(key string) error {
	log.Printf("模拟按键: %s", key)
	return nil
}

// PressKeyWithModifiers 模拟按下带修饰键的按键
func (ki *KeyboardInput) PressKeyWithModifiers(key string, modifiers ...string) error {
	log.Printf("模拟按键: %s 带修饰键: %v", key, modifiers)
	return nil
}

// GetActiveWindow 获取当前活动窗口
func (ki *KeyboardInput) GetActiveWindow() string {
	// 模拟获取当前活动窗口的标题
	return "模拟窗口"
}

// FocusWindow 聚焦到指定窗口
func (ki *KeyboardInput) FocusWindow(title string) error {
	log.Printf("模拟聚焦窗口: %s", title)
	return nil
}

// TypeWithDelay 以指定的延迟输入文本（每个字符之间有延迟）
func (ki *KeyboardInput) TypeWithDelay(text string, delayMS int) error {
	log.Printf("模拟延迟输入: %s (延迟: %dms)", text, delayMS)
	return nil
}

// PasteText 粘贴文本（使用剪贴板）
func (ki *KeyboardInput) PasteText(text string) error {
	log.Printf("模拟粘贴文本: %s", text)
	return nil
} 