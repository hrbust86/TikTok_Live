package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketClient WebSocket客户端结构体
type WebSocketClient struct {
	conn    *websocket.Conn
	url     string
	headers map[string]string
}

// NewWebSocketClient 创建新的WebSocket客户端
func NewWebSocketClient(serverURL string, headers map[string]string) *WebSocketClient {
	return &WebSocketClient{
		url:     serverURL,
		headers: headers,
	}
}

// Connect 连接到WebSocket服务器
func (c *WebSocketClient) Connect() error {
	u, err := url.Parse(c.url)
	if err != nil {
		return fmt.Errorf("解析URL失败: %v", err)
	}

	// 创建WebSocket连接
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	// 设置请求头
	requestHeaders := make(map[string][]string)
	for key, value := range c.headers {
		requestHeaders[key] = []string{value}
	}

	conn, _, err := dialer.Dial(u.String(), requestHeaders)
	if err != nil {
		return fmt.Errorf("连接WebSocket服务器失败: %v", err)
	}

	c.conn = conn
	log.Printf("成功连接到WebSocket服务器: %s", c.url)
	return nil
}

// SendMessage 发送文本消息
func (c *WebSocketClient) SendMessage(message string) error {
	if c.conn == nil {
		return fmt.Errorf("WebSocket连接未建立")
	}

	err := c.conn.WriteMessage(websocket.TextMessage, []byte(message))
	if err != nil {
		return fmt.Errorf("发送消息失败: %v", err)
	}

	log.Printf("成功发送文本消息: %s", message)
	return nil
}

// SendBinary 发送二进制数据
func (c *WebSocketClient) SendBinary(data []byte) error {
	if c.conn == nil {
		return fmt.Errorf("WebSocket连接未建立")
	}

	err := c.conn.WriteMessage(websocket.BinaryMessage, data)
	if err != nil {
		return fmt.Errorf("发送二进制数据失败: %v", err)
	}

	log.Printf("成功发送二进制数据，长度: %d bytes", len(data))
	return nil
}

// SendJSON 发送JSON数据
func (c *WebSocketClient) SendJSON(data interface{}) error {
	if c.conn == nil {
		return fmt.Errorf("WebSocket连接未建立")
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("JSON序列化失败: %v", err)
	}

	err = c.conn.WriteMessage(websocket.TextMessage, jsonData)
	if err != nil {
		return fmt.Errorf("发送JSON消息失败: %v", err)
	}

	log.Printf("成功发送JSON消息: %s", string(jsonData))
	return nil
}

// SendMarshalData 发送marshal数据（protobuf序列化后的数据）
func (c *WebSocketClient) SendMarshalData(marshalData []byte) error {
	if c.conn == nil {
		return fmt.Errorf("WebSocket连接未建立")
	}

	// 将marshal数据作为二进制消息发送
	err := c.conn.WriteMessage(websocket.BinaryMessage, marshalData)
	if err != nil {
		return fmt.Errorf("发送marshal数据失败: %v", err)
	}

	log.Printf("成功发送marshal数据，长度: %d bytes", len(marshalData))
	return nil
}

// Listen 监听服务器消息
func (c *WebSocketClient) Listen() {
	if c.conn == nil {
		log.Println("WebSocket连接未建立，无法监听")
		return
	}

	log.Println("开始监听服务器消息...")
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket连接异常关闭: %v", err)
			}
			break
		}

		log.Printf("收到服务器消息: %s", string(message))
	}
}

// Close 关闭WebSocket连接
func (c *WebSocketClient) Close() error {
	if c.conn == nil {
		return nil
	}

	err := c.conn.Close()
	if err != nil {
		return fmt.Errorf("关闭WebSocket连接失败: %v", err)
	}

	log.Println("WebSocket连接已关闭")
	return nil
}

// IsConnected 检查连接状态
func (c *WebSocketClient) IsConnected() bool {
	return c.conn != nil
}

// Reconnect 重新连接
func (c *WebSocketClient) Reconnect() error {
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}

	return c.Connect()
}
