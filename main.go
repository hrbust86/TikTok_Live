package main

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"google.golang.org/protobuf/reflect/protoreflect"

	tiktok_hack "Sunny/tiktok_hack/generated"

	"sync/atomic"

	"github.com/qtgolang/SunnyNet/SunnyNet"
	"github.com/qtgolang/SunnyNet/src/public"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var (
	Sunny = SunnyNet.NewSunny()

	// WebSocket客户端实例
	wsClient      *WebSocketClient
	wsClientMutex sync.Mutex

	// 消息队列相关
	messageQueue = make(chan MessageData, 10000) // 缓冲10000条消息
	queueClosed  = make(chan struct{})
)

// 消息数据结构
type MessageData struct {
	Method      string
	MarshalData []byte
	Timestamp   int64
	SequenceID  uint64 // 序列号，确保顺序
}

// 全局序列号计数器
var sequenceCounter uint64 = 0

// 获取下一个序列号
func getNextSequenceID() uint64 {
	return atomic.AddUint64(&sequenceCounter, 1)
}

// 初始化WebSocket客户端
func initWebSocketClient() {
	wsClientMutex.Lock()
	defer wsClientMutex.Unlock()

	// 从配置文件获取WebSocket服务器配置
	config := LoadConfig()

	// 创建WebSocket客户端
	wsClient = NewWebSocketClient(config.WebSocketServer.URL, config.WebSocketServer.Headers)

	// 连接到WebSocket服务器
	if err := wsClient.Connect(); err != nil {
		log.Printf("连接WebSocket服务器失败: %v", err)
		return
	}

	// 启动监听协程
	go wsClient.Listen()

	log.Println("WebSocket客户端初始化完成")
}

func main() {
	// 检查命令行参数
	if len(os.Args) > 1 && os.Args[1] == "test" {
		// 运行测试模式
		RunTests()
		return
	}

	// 启动消息发送协程
	startMessageSender()

	// 初始化WebSocket客户端
	initWebSocketClient()

	// 绑定回调函数
	Sunny.SetGoCallback(HttpCallback, TcpCallback, WSCallback, UdpCallback)

	// 加载配置
	config := LoadConfig()

	// 设置端口并启动 SunnyNet 代理服务器
	s := Sunny.SetPort(config.TikTokProxy.Port)
	defer s.Close()
	//随机tls指纹
	//s.SetRandomTLS(true)
	s.SetGlobalProxy(config.TikTokProxy.UpstreamProxy, config.TikTokProxy.Timeout)
	st := s.Start()
	if st.Error != nil {
		log.Fatalf(st.Error.Error())
	}

	fmt.Printf("浏览器代理设置为:127.0.0.1:%d\n", config.TikTokProxy.Port)
	fmt.Printf("上游代理地址为:%s\n", config.TikTokProxy.UpstreamProxy)
	fmt.Printf("WebSocket 服务地址为:%s\n", GetWebSocketServerURL())
	fmt.Println("正在运行....")

	// 避免程序退出
	select {}
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

			// 发送marshal数据到WebSocket服务器
			if v.Method == "WebcastGiftMessage" || v.Method == "WebcastChatMessage" {
				sendMarshalDataToWebSocket(v.Method, marshal)
			}
			if v.Method == "WebcastGiftMessage" {
				//processGiftMessage(marshal)
			}

			if v.Method == "WebcastChatMessage" {
				//	processChatMessage(marshal)
			}

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

// sendMarshalDataToWebSocket 将marshal数据异步发送到WebSocket服务器（保证顺序）
func sendMarshalDataToWebSocket(method string, marshalData []byte) {
	// 创建消息数据
	msg := MessageData{
		Method:      method,
		MarshalData: marshalData,
		Timestamp:   time.Now().Unix(),
		SequenceID:  getNextSequenceID(),
	}

	// 异步将消息加入队列，不阻塞WSCallback
	go func() {
		select {
		case messageQueue <- msg:
			// 消息成功加入队列
			log.Printf("消息已加入队列: %s (序列号: %d)", method, msg.SequenceID)
		case <-queueClosed:
			// 队列已关闭
			log.Printf("消息队列已关闭，丢弃消息: %s", method)
		default:
			// 队列满了，记录警告
			log.Printf("警告: 消息队列已满，丢弃消息: %s (序列号: %d)", method, msg.SequenceID)
		}
	}()
}

// startMessageSender 启动消息发送协程
func startMessageSender() {
	go func() {
		log.Println("启动消息发送协程...")

		for {
			select {
			case msg, ok := <-messageQueue:
				if !ok {
					log.Println("消息队列已关闭，发送协程退出")
					return
				}

				// 发送消息到WebSocket服务器
				if err := sendMessageToWebSocket(msg); err != nil {
					log.Printf("发送消息失败 (序列号: %d): %v", msg.SequenceID, err)
				} else {
					log.Printf("消息发送成功 (序列号: %d): %s", msg.SequenceID, msg.Method)
				}

			case <-queueClosed:
				log.Println("收到关闭信号，发送协程退出")
				return
			}
		}
	}()
}

// sendMessageToWebSocket 实际发送消息到WebSocket服务器
func sendMessageToWebSocket(msg MessageData) error {
	wsClientMutex.Lock()
	defer wsClientMutex.Unlock()

	// 检查WebSocket客户端状态
	if wsClient == nil || !wsClient.IsConnected() {
		log.Printf("WebSocket客户端未连接，尝试重新连接... (序列号: %d)", msg.SequenceID)
		// 异步重连
		go func() {
			initWebSocketClient()
		}()
		return fmt.Errorf("WebSocket客户端未连接")
	}

	// 创建发送消息结构
	// 将marshalData转换为字符串
	dataString := string(msg.MarshalData)
	message := map[string]interface{}{
		"method":      msg.Method,
		"data":        dataString,
		"timestamp":   msg.Timestamp,
		"sequence_id": msg.SequenceID,
	}

	// 发送JSON消息到WebSocket服务器
	if err := wsClient.SendJSON(message); err != nil {
		log.Printf("发送消息到WebSocket服务器失败 (序列号: %d): %v", msg.SequenceID, err)
		// 异步重连
		go func() {
			if err := wsClient.Reconnect(); err != nil {
				log.Printf("重新连接失败: %v", err)
			}
		}()
		return err
	}

	return nil
}

// 优雅关闭消息队列
func shutdownMessageQueue() {
	close(queueClosed)
	close(messageQueue)
	log.Println("消息队列已关闭")
}
