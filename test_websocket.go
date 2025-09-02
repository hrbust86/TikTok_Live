package main

import (
	"fmt"
	"time"
)

// TestWebSocketClient 测试WebSocket客户端功能
func TestWebSocketClient() {
	fmt.Println("=== WebSocket客户端测试 ===")

	// 测试配置加载
	fmt.Println("1. 测试配置加载...")
	config := LoadConfig()
	fmt.Printf("   WebSocket服务器URL: %s\n", config.WebSocketServer.URL)
	fmt.Printf("   TikTok代理端口: %d\n", config.TikTokProxy.Port)
	fmt.Printf("   上游代理: %s\n", config.TikTokProxy.UpstreamProxy)

	// 使用配置文件中的WebSocket服务器地址
	fmt.Println("\n2. 创建WebSocket客户端...")
	client := NewWebSocketClient(config.WebSocketServer.URL, config.WebSocketServer.Headers)

	if client == nil {
		fmt.Println("   ❌ 客户端创建失败")
		return
	}
	fmt.Println("   ✅ 客户端创建成功")

	// 连接到WebSocket服务器
	fmt.Println("\n3. 连接到WebSocket服务器...")
	err := client.Connect()
	if err != nil {
		fmt.Printf("   ❌ 连接失败: %v\n", err)
		fmt.Println("   请确保WebSocket服务器正在运行")
		fmt.Printf("   可以运行: go run websocket_server.go 来启动测试服务器\n")
		return
	}
	fmt.Println("   ✅ 连接成功")

	// 启动监听协程
	go client.Listen()

	// 等待一下确保连接稳定
	time.Sleep(1 * time.Second)

	for {
		// 发送testData
		fmt.Println("\n4. 发送testData...")
		testData := map[string]interface{}{
			"method":    "TestMessage",
			"data":      "Hello, WebSocket!",
			"timestamp": time.Now().Unix(),
			"test":      true,
			"message":   "这是一条测试消息，通过WebSocket连接发送",
		}

		fmt.Printf("   📤 准备发送消息: %+v\n", testData)

		// 实际发送JSON消息到WebSocket服务器
		if err := client.SendJSON(testData); err != nil {
			fmt.Printf("   ❌ 发送消息失败: %v\n", err)
		} else {
			fmt.Println("   ✅ 消息发送成功")
		}

		// 发送更多测试数据
		fmt.Println("\n5. 发送更多测试数据...")

		// 模拟TikTok弹幕消息
		chatMessage := map[string]interface{}{
			"method":    "WebcastChatMessage",
			"data":      "用户发送了一条聊天消息",
			"timestamp": time.Now().Unix(),
			"user": map[string]interface{}{
				"nickname": "测试用户",
				"userId":   "12345",
			},
			"content": "你好，这是一条测试聊天消息！",
		}

		if err := client.SendJSON(chatMessage); err != nil {
			fmt.Printf("   ❌ 发送聊天消息失败: %v\n", err)
		} else {
			fmt.Println("   ✅ 聊天消息发送成功")
		}

		// 模拟礼物消息
		giftMessage := map[string]interface{}{
			"method":    "WebcastGiftMessage",
			"data":      "用户送出了礼物",
			"timestamp": time.Now().Unix(),
			"user": map[string]interface{}{
				"nickname": "礼物用户",
				"userId":   "67890",
			},
			"gift": map[string]interface{}{
				"name":  "测试礼物",
				"value": 100,
			},
		}

		if err := client.SendJSON(giftMessage); err != nil {
			fmt.Printf("   ❌ 发送礼物消息失败: %v\n", err)
		} else {
			fmt.Println("   ✅ 礼物消息发送成功")
		}

		// 等待一段时间让消息发送完成
		fmt.Println("\n6. 等待消息发送完成...")
		time.Sleep(3 * time.Second)
	}
	// 关闭连接
	//fmt.Println("\n7. 关闭WebSocket连接...")
	//if err := client.Close(); err != nil {
	//	fmt.Printf("   ⚠️  关闭连接时出现警告: %v\n", err)
	//} else {
	//	fmt.Println("   ✅ 连接已关闭")
	//}
	select {}
	//fmt.Println("\n=== 测试完成 ===")
}

// TestConfigUpdate 测试配置更新功能
func TestConfigUpdate() {
	fmt.Println("\n=== 配置更新测试 ===")

	// 测试更新WebSocket服务器配置
	fmt.Println("1. 测试配置更新...")
	oldURL := GetWebSocketServerURL()
	fmt.Printf("   当前WebSocket服务器URL: %s\n", oldURL)

	// 更新配置
	newURL := "ws://localhost:9090/ws"
	newHeaders := map[string]string{
		"User-Agent":    "UpdatedClient/1.0",
		"Authorization": "Bearer new-token",
	}

	UpdateWebSocketServer(newURL, newHeaders)

	// 验证更新
	updatedURL := GetWebSocketServerURL()
	updatedHeaders := GetWebSocketServerHeaders()

	fmt.Printf("   更新后WebSocket服务器URL: %s\n", updatedURL)
	fmt.Printf("   更新后请求头: %+v\n", updatedHeaders)

	if updatedURL == newURL {
		fmt.Println("   ✅ URL更新成功")
	} else {
		fmt.Println("   ❌ URL更新失败")
	}

	// 恢复原配置
	UpdateWebSocketServer(oldURL, nil)
	fmt.Println("   已恢复原配置")

	fmt.Println("=== 配置更新测试完成 ===")
}

// RunTests 运行所有测试
func RunTests() {
	fmt.Println("开始运行WebSocket客户端测试...\n")

	TestWebSocketClient()
	//TestConfigUpdate()

	fmt.Println("\n所有测试完成！")
}
