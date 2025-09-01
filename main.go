package main

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"

	"google.golang.org/protobuf/reflect/protoreflect"

	tiktok_hack "Sunny/tiktok_hack/generated"

	"github.com/qtgolang/SunnyNet/SunnyNet"
	"github.com/qtgolang/SunnyNet/src/public"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var Sunny = SunnyNet.NewSunny()

//var agentlist sync.Map // 使用 sync.Map 替代 map[string]*agent

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

	fmt.Println("浏览器代理设置为:127.0.0.1:23809")
	fmt.Println("上游代理地址为:127.0.0.1:21586")
	//fmt.Println("WebSocket 服务地址为:ws://127.0.0.1:18080/ws")
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

			if v.Method == "WebcastGiftMessage" {
				processGiftMessage(marshal)
			}

			if v.Method == "WebcastChatMessage" {
				processChatMessage(marshal)
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
