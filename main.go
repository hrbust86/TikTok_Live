package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"google.golang.org/protobuf/reflect/protoreflect"

	tiktok_hack "Sunny/tiktok_hack/generated"

	"github.com/gorilla/websocket"
	"github.com/qtgolang/SunnyNet/SunnyNet"
	"github.com/qtgolang/SunnyNet/src/public"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var Sunny = SunnyNet.NewSunny()
var agentlist sync.Map // 使用 sync.Map 替代 map[string]*agent

// agent 结构体用于存储 WebSocket 连接及其相关信息
func main() {
	// 绑定回调函数
	Sunny.SetGoCallback(HttpCallback, TcpCallback, WSCallback, UdpCallback)

	// 设置端口并启动 SunnyNet 代理服务器
	s := Sunny.SetPort(23809)
	defer s.Close()
	//随机tls指纹
	//s.SetRandomTLS(true)
	s.SetGlobalProxy("socket5://127.0.0.1:21586", 60000)
	st := s.Start()
	if st.Error != nil {
		log.Fatalf(st.Error.Error())
	}

	// 设置 WebSocket 升级器
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // 允许所有 CORS 请求
		},
	}

	// 处理 WebSocket 请求
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		handleWebSocket(upgrader, w, r)
	})

	fmt.Println("浏览器代理设置为:127.0.0.1:23809")
	fmt.Println("上游代理地址为:127.0.0.1:21586")
	fmt.Println("WebSocket 服务地址为:ws://127.0.0.1:18080/ws")
	fmt.Println("正在运行....")

	// 启动 HTTP 服务器（用于 WebSocket）
	go func() {
		err := http.ListenAndServe(":18080", nil)
		if err != nil {
			log.Fatalf("ListenAndServe: %v", err)
		}
	}()

	log.Println("WebSocket 服务启动成功.ws端口为18080")

	// 避免程序退出
	select {}
}

// handleWebSocket 处理 WebSocket 连接
func handleWebSocket(upgrader websocket.Upgrader, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket 升级失败:", err)
		return
	}
	defer conn.Close()

	key := r.Header.Get("Sec-WebSocket-Key")
	if key == "" {
		log.Println("缺少 Sec-WebSocket-Key 头")
		return
	}

	agent := &agent{
		queue: make(chan []byte, 2),
		stop:  make(chan struct{}),
		conn:  conn,
	}
	agentlist.Store(key, agent)
	go handleClient(agent)

	log.Println("当前连接数", getAgentCount())

	// 设置读取超时时间
	err = agent.conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	if err != nil {
		log.Println("设置读取超时失败:", err)
		return
	}

	defer func() {
		log.Println(key, "断开连接")
		close(agent.stop)
		agentlist.Delete(key)
	}()

	for {
		mt, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("读取消息失败:", err)
			break
		}
		log.Printf("收到消息: %s", message)
		if string(message) == "ping" {
			if err := conn.WriteMessage(mt, []byte("pong")); err != nil {
				log.Println("发送消息失败:", err)
				break
			} else {
				_ = agent.conn.SetReadDeadline(time.Now().Add(30 * time.Second))
			}
		}
	}
}

// handleClient 处理客户端消息
func handleClient(a *agent) {
	for {
		select {
		case <-a.stop:
			return
		case b := <-a.queue:
			err := a.conn.WriteMessage(websocket.TextMessage, b)
			if err != nil {
				log.Println("发送消息失败:", err)
			}
		}
	}
}

// getAgentCount 获取当前连接数
func getAgentCount() int {
	count := 0
	agentlist.Range(func(_, _ interface{}) bool {
		count++
		return true
	})
	return count
}

// HttpCallback HTTP 回调函数
func HttpCallback(Conn SunnyNet.ConnHTTP) {
	// 处理 HTTP 连接
}

// WSCallback WebSocket 回调函数
func WSCallback(Conn SunnyNet.ConnWebSocket) {
	if !strings.Contains(Conn.URL(), "tiktok.com/webcast/im/") {
		return
	}

	message := Conn.Body()
	PushFrame := &tiktok_hack.WebcastPushFrame{}
	err := proto.Unmarshal(message, PushFrame)
	if err != nil {
		log.Println("解析消息失败:", err)
		return
	}

	if PushFrame.PayloadType == "ack" {
		return // 心跳包数据不处理
	}

	isGzip := CheckGzip(PushFrame)
	if isGzip && PushFrame.PayloadType == "msg" {
		gzipReader, err := gzip.NewReader(bytes.NewReader(PushFrame.Payload))
		if err != nil {
			log.Println("解析 Gzip 消息失败:", err)
			return
		}
		defer gzipReader.Close()

		uncompressedData, err := io.ReadAll(gzipReader)
		if err != nil {
			log.Println("读取解压数据失败:", err)
			return
		}

		response := &tiktok_hack.WebcastResponse{}
		err = proto.Unmarshal(uncompressedData, response)
		if err != nil {
			log.Println("解析解压数据失败:", err)
			return
		}

		for _, v := range response.Messages {
			msg, err := MatchMethod(v.Method)
			if err != nil {
				//log.Printf("未知消息，无法处理: %v, %s\n", err, hex.EncodeToString(v.Payload))
				continue
			}
			err = proto.Unmarshal(v.Payload, msg)
			if err != nil {
				//log.Println("解析消息失败:", err)
				continue
			}

			// 序列化为 JSON
			marshal, err := protojson.Marshal(msg)
			if err != nil {
				log.Println("JSON 序列化失败:", err)
				continue
			}

			if v.Method == "WebcastGiftMessage" {
				processGiftMessage(marshal)
				//log.Println(string(marshal) + "\n\n")

				// 追加写入到文件
				//err = appendToFile("gift_messages.json", string(marshal)+"\n\n")
				//if err != nil {
				//	log.Printf("写入文件失败: %v", err)
				//}
			}

			if v.Method == "WebcastChatMessage" {
				//log.Println(string(marshal))
			}

			// 遍历 agentlist，将消息发送到每个连接
			agentlist.Range(func(_, value interface{}) bool {
				agent := value.(*agent)
				agent.queue <- marshal
				return true
			})
		}

	}
}

// CheckGzip 检查协议头当中是否包含gzip
func CheckGzip(headers *tiktok_hack.WebcastPushFrame) bool {
	return headers.Headers["compress_type"] == "gzip"
}

// TcpCallback TCP 回调函数
func TcpCallback(Conn SunnyNet.ConnTCP) {
	// 处理 TCP 连接
}

// UdpCallback UDP 回调函数
func UdpCallback(Conn SunnyNet.ConnUDP) {
	if public.SunnyNetUDPTypeReceive == Conn.Type() {
		// 处理接收的 UDP 数据
	}
	if public.SunnyNetUDPTypeSend == Conn.Type() {
		// 处理发送的 UDP 数据
	}
	if public.SunnyNetUDPTypeClosed == Conn.Type() {
		// 处理关闭的 UDP 连接
	}
}
func MatchMethod(method string) (protoreflect.ProtoMessage, error) {
	if createMessage, ok := messageTypeMap[method]; ok {
		return createMessage(), nil
	}
	return nil, errors.New("未知消息: " + method)
}

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
