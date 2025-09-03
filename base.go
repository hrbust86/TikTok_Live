package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

// processGiftMessage 处理礼物消息
func processGiftMessage(marshal []byte) {
	// 解析JSON以获取giftId
	var jsonData map[string]interface{}
	if err := json.Unmarshal(marshal, &jsonData); err == nil {
		if giftId, exists := jsonData["giftId"]; exists {
			log.Printf("giftId: %v", giftId)
			log.Printf("fanTicketCount: %v", jsonData["fanTicketCount"])
			log.Printf("groupCount: %v", jsonData["groupCount"])
			log.Printf("repeatCount: %v", jsonData["repeatCount"])
			log.Printf("comboCount: %v", jsonData["comboCount"])
			log.Printf("repeatEnd: %v", jsonData["repeatEnd"])

			// 解析gift对象下的name字段
			if gift, exists := jsonData["gift"]; exists {
				if giftMap, ok := gift.(map[string]interface{}); ok {
					if name, exists := giftMap["name"]; exists {
						log.Printf("礼物名称: %v", name)
						log.Printf("combo: %v", giftMap["combo"])
					} else {
						log.Println("未找到礼物名称字段")
					}
				}
			} else {
				log.Println("未找到gift字段")
			}

			// 解析user对象下的nickname字段
			if user, exists := jsonData["user"]; exists {
				if userMap, ok := user.(map[string]interface{}); ok {
					if nickname, exists := userMap["nickname"]; exists {
						log.Printf("用户昵称: %v", nickname)
					} else {
						log.Println("未找到用户昵称字段")
					}
				} else {
					log.Println("user字段不是对象类型")
				}
			} else {
				log.Println("未找到user字段")
			}
		} else {
			log.Println("未找到giftId字段")
		}
		log.Println("--------------------------------")
	} else {
		log.Printf("解析JSON失败: %v", err)
	}
}

// processChatMessage 处理聊天消息
func processChatMessage(marshal []byte) {
	// 解析WebcastChatMessage中的user和content字段
	var jsonData map[string]interface{}
	if err := json.Unmarshal(marshal, &jsonData); err == nil {
		// 解析user对象下的字段
		if user, exists := jsonData["user"]; exists {
			if userMap, ok := user.(map[string]interface{}); ok {
				// 解析user下的nickname字段
				if nickname, exists := userMap["nickname"]; exists {
					log.Printf("用户昵称: %v", nickname)
				}
				// 解析user下的其他字段
				if avatarThumb, exists := userMap["avatarThumb"]; exists {
					log.Printf("用户头像: %v", avatarThumb)
				}
			}
		}

		// 解析content字段（这是消息内容，不是user下的字段）
		if content, exists := jsonData["content"]; exists {
			log.Printf("消息内容: %v", content)
		}
		log.Println("--------------------------------")
	}
}

// appendToFile 追加内容到文件
func appendToFile(filename, content string) error {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(content)
	return err
}

// 日志管理相关常量和变量
const (
	MaxLogFileSize = 10 * 1024 * 1024 // 10MB 最大日志文件大小
	MaxLogFiles    = 2                // 最大日志文件数量
)

var (
	logMutex sync.Mutex
	logFile  *os.File
	logSize  int64
)

// LogError 错误日志记录函数
// 既能控制台输出也能记录到本地日志
// 自动管理日志文件大小和数量
func LogError(format string, args ...interface{}) {
	// 控制台输出
	log.Printf(format, args...)

	// 本地日志记录
	logMutex.Lock()
	defer logMutex.Unlock()

	// 确保日志文件已打开
	if err := ensureLogFile(); err != nil {
		log.Printf("无法创建日志文件: %v", err)
		return
	}

	// 格式化日志消息
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logMessage := fmt.Sprintf("[%s] ERROR: %s\n", timestamp, fmt.Sprintf(format, args...))

	// 写入日志文件
	if _, err := logFile.WriteString(logMessage); err != nil {
		log.Printf("写入日志文件失败: %v", err)
		return
	}

	// 更新文件大小
	logSize += int64(len(logMessage))

	// 检查是否需要轮转日志文件
	if logSize >= MaxLogFileSize {
		rotateLogFile()
	}
}

// ensureLogFile 确保日志文件已打开
func ensureLogFile() error {
	if logFile != nil {
		return nil
	}

	// 创建logs目录
	if err := os.MkdirAll("logs", 0755); err != nil {
		return err
	}

	// 打开主日志文件
	filename := "logs/error.log"
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	// 获取文件大小
	stat, err := file.Stat()
	if err != nil {
		file.Close()
		return err
	}

	logFile = file
	logSize = stat.Size()

	return nil
}

// rotateLogFile 轮转日志文件
func rotateLogFile() {
	if logFile == nil {
		return
	}

	// 关闭当前日志文件
	logFile.Close()
	logFile = nil

	// 重命名现有日志文件
	oldFile := "logs/error.log"
	newFile := "logs/error.old.log"

	// 如果旧文件存在，先删除
	if _, err := os.Stat(newFile); err == nil {
		os.Remove(newFile)
	}

	// 重命名当前文件
	if _, err := os.Stat(oldFile); err == nil {
		os.Rename(oldFile, newFile)
	}

	// 重置日志大小
	logSize = 0

	log.Printf("日志文件已轮转: %s -> %s", oldFile, newFile)
}

// LogInfo 信息日志记录函数
func LogInfo(format string, args ...interface{}) {
	// 控制台输出
	log.Printf(format, args...)

	// 本地日志记录
	logMutex.Lock()
	defer logMutex.Unlock()

	// 确保日志文件已打开
	if err := ensureLogFile(); err != nil {
		log.Printf("无法创建日志文件: %v", err)
		return
	}

	// 格式化日志消息
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logMessage := fmt.Sprintf("[%s] INFO: %s\n", timestamp, fmt.Sprintf(format, args...))

	// 写入日志文件
	if _, err := logFile.WriteString(logMessage); err != nil {
		log.Printf("写入日志文件失败: %v", err)
		return
	}

	// 更新文件大小
	logSize += int64(len(logMessage))

	// 检查是否需要轮转日志文件
	if logSize >= MaxLogFileSize {
		rotateLogFile()
	}
}

// LogWarning 警告日志记录函数
func LogWarning(format string, args ...interface{}) {
	// 控制台输出
	log.Printf(format, args...)

	// 本地日志记录
	logMutex.Lock()
	defer logMutex.Unlock()

	// 确保日志文件已打开
	if err := ensureLogFile(); err != nil {
		log.Printf("无法创建日志文件: %v", err)
		return
	}

	// 格式化日志消息
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logMessage := fmt.Sprintf("[%s] WARNING: %s\n", timestamp, fmt.Sprintf(format, args...))

	// 写入日志文件
	if _, err := logFile.WriteString(logMessage); err != nil {
		log.Printf("写入日志文件失败: %v", err)
		return
	}

	// 更新文件大小
	logSize += int64(len(logMessage))

	// 检查是否需要轮转日志文件
	if logSize >= MaxLogFileSize {
		rotateLogFile()
	}
}

// CloseLogFile 关闭日志文件
func CloseLogFile() {
	logMutex.Lock()
	defer logMutex.Unlock()

	if logFile != nil {
		logFile.Close()
		logFile = nil
		logSize = 0
	}
}
