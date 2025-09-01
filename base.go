package main

import (
	"encoding/json"
	"log"
	"os"
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
