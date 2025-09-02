#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
WebSocket服务器，支持ping pong检测
Python 3.8兼容版本
"""

import asyncio
import json
import logging
import time
from typing import Dict, Set, Optional
import websockets
from websockets.server import WebSocketServerProtocol

# 配置日志
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

class WebSocketServer:
    def __init__(self, host: str = "localhost", port: int = 8765):
        self.host = host
        self.port = port
        self.clients: Set[WebSocketServerProtocol] = set()
        self.client_info: Dict[WebSocketServerProtocol, Dict] = {}
        self.ping_interval = 30  # 30秒发送一次ping
        self.pong_timeout = 10   # 10秒内必须收到pong响应
        self.max_missed_pongs = 3  # 最多允许丢失3次pong响应
        
    async def register_client(self, websocket: WebSocketServerProtocol):
        """注册新客户端"""
        self.clients.add(websocket)
        self.client_info[websocket] = {
            'connected_at': time.time(),
            'last_pong': time.time(),
            'missed_pongs': 0,
            'client_id': f"client_{len(self.clients)}"
        }
        logger.info(f"新客户端连接: {self.client_info[websocket]['client_id']}")
        
    async def unregister_client(self, websocket: WebSocketServerProtocol):
        """注销客户端"""
        if websocket in self.clients:
            client_id = self.client_info[websocket]['client_id']
            self.clients.remove(websocket)
            del self.client_info[websocket]
            logger.info(f"客户端断开连接: {client_id}")
            
    async def send_ping(self, websocket: WebSocketServerProtocol):
        """发送ping消息"""
        try:
            ping_message = {
                "type": "ping",
                "timestamp": time.time(),
                "data": "ping"
            }
            await websocket.send(json.dumps(ping_message))
            logger.debug(f"发送ping到客户端: {self.client_info[websocket]['client_id']}")
        except Exception as e:
            logger.error(f"发送ping失败: {e}")
            await self.unregister_client(websocket)
            
    async def handle_pong(self, websocket: WebSocketServerProtocol, data: dict):
        """处理pong响应"""
        if websocket in self.client_info:
            self.client_info[websocket]['last_pong'] = time.time()
            self.client_info[websocket]['missed_pongs'] = 0
            logger.debug(f"收到pong响应: {self.client_info[websocket]['client_id']}")
            
    async def check_pong_timeout(self):
        """检查pong超时"""
        current_time = time.time()
        disconnected_clients = []
        
        for websocket, info in self.client_info.items():
            time_since_last_pong = current_time - info['last_pong']
            
            if time_since_last_pong > self.pong_timeout:
                info['missed_pongs'] += 1
                logger.warning(f"客户端 {info['client_id']} 丢失pong响应 #{info['missed_pongs']}")
                
                if info['missed_pongs'] >= self.max_missed_pongs:
                    logger.error(f"客户端 {info['client_id']} 超过最大丢失pong次数，断开连接")
                    disconnected_clients.append(websocket)
                    
        # 断开超时的客户端
        for websocket in disconnected_clients:
            await self.unregister_client(websocket)
            
    async def ping_clients(self):
        """定期向所有客户端发送ping"""
        while True:
            try:
                await asyncio.sleep(self.ping_interval)
                
                if not self.clients:
                    continue
                    
                # 检查pong超时
                await self.check_pong_timeout()
                
                # 向所有客户端发送ping
                ping_tasks = []
                for websocket in list(self.clients):
                    if websocket.open:
                        ping_tasks.append(self.send_ping(websocket))
                        
                if ping_tasks:
                    await asyncio.gather(*ping_tasks, return_exceptions=True)
                    
            except Exception as e:
                logger.error(f"Ping任务出错: {e}")
                
    async def handle_message(self, websocket: WebSocketServerProtocol, message: str):
        """处理客户端消息"""
        try:
            data = json.loads(message)
            message_type = data.get('type', 'unknown')
            
            # 记录收到的消息
            logger.info(f"收到消息类型: {message_type}, 内容: {data}")
            
            if message_type == 'pong':
                await self.handle_pong(websocket, data)
            elif message_type == 'chat':
                # 处理聊天消息
                await self.broadcast_message(data, exclude_websocket=websocket)
            elif message_type == 'join':
                # 处理加入消息
                await self.broadcast_message({
                    'type': 'system',
                    'message': f"用户 {data.get('username', 'Unknown')} 加入了聊天室",
                    'timestamp': time.time()
                })
            elif message_type == 'broadcast':
                # 处理广播消息
                await self.broadcast_message(data, exclude_websocket=websocket)
            elif message_type == 'private':
                # 处理私聊消息
                await self.handle_private_message(websocket, data)
            elif message_type == 'status':
                # 处理状态更新消息
                await self.handle_status_update(websocket, data)
            elif message_type == 'command':
                # 处理命令消息
                await self.handle_command(websocket, data)
            elif message_type == 'file':
                # 处理文件相关消息
                await self.handle_file_message(websocket, data)
            elif message_type == 'notification':
                # 处理通知消息
                await self.handle_notification(websocket, data)
            elif message_type == 'heartbeat':
                # 处理心跳消息
                await self.handle_heartbeat(websocket, data)
            elif message_type == 'room':
                # 处理房间相关消息
                await self.handle_room_message(websocket, data)
            elif message_type == 'user':
                # 处理用户相关消息
                await self.handle_user_message(websocket, data)
            else:
                # 处理任意其他类型的消息
                await self.handle_arbitrary_message(websocket, data)
                
        except json.JSONDecodeError:
            logger.warning(f"收到无效JSON消息: {message}")
        except Exception as e:
            logger.error(f"处理消息时出错: {e}")
            
    async def handle_private_message(self, websocket: WebSocketServerProtocol, data: dict):
        """处理私聊消息"""
        target_user_id = data.get('target_user_id')
        if target_user_id:
            # 查找目标用户并发送私聊消息
            for client_ws in self.clients:
                if (client_ws != websocket and 
                    self.client_info[client_ws].get('client_id') == target_user_id):
                    try:
                        private_msg = {
                            'type': 'private',
                            'from_user': self.client_info[websocket]['client_id'],
                            'message': data.get('message', ''),
                            'timestamp': time.time()
                        }
                        await client_ws.send(json.dumps(private_msg))
                        logger.info(f"私聊消息已发送: {data.get('message', '')[:50]}...")
                    except Exception as e:
                        logger.error(f"发送私聊消息失败: {e}")
                    break
            else:
                # 目标用户不在线，发送离线通知
                offline_notice = {
                    'type': 'system',
                    'message': f"用户 {target_user_id} 不在线，私聊消息发送失败",
                    'timestamp': time.time()
                }
                await websocket.send(json.dumps(offline_notice))
        else:
            logger.warning("私聊消息缺少目标用户ID")
            
    async def handle_status_update(self, websocket: WebSocketServerProtocol, data: dict):
        """处理状态更新消息"""
        if websocket in self.client_info:
            # 更新客户端状态信息
            status = data.get('status', {})
            self.client_info[websocket].update(status)
            
            # 广播状态更新
            status_msg = {
                'type': 'status_update',
                'user_id': self.client_info[websocket]['client_id'],
                'status': status,
                'timestamp': time.time()
            }
            await self.broadcast_message(status_msg, exclude_websocket=websocket)
            logger.info(f"用户 {self.client_info[websocket]['client_id']} 状态已更新")
            
    async def handle_command(self, websocket: WebSocketServerProtocol, data: dict):
        """处理命令消息"""
        command = data.get('command', '')
        params = data.get('params', {})
        
        if command == 'get_users':
            # 获取在线用户列表
            users_list = [
                {
                    'client_id': info['client_id'],
                    'connected_at': info['connected_at'],
                    'status': info.get('status', {})
                }
                for info in self.client_info.values()
            ]
            response = {
                'type': 'command_response',
                'command': command,
                'result': users_list,
                'timestamp': time.time()
            }
            await websocket.send(json.dumps(response))
            
        elif command == 'get_stats':
            # 获取服务器统计信息
            stats = {
                'total_clients': len(self.clients),
                'server_uptime': time.time() - getattr(self, 'start_time', time.time()),
                'ping_interval': self.ping_interval,
                'pong_timeout': self.pong_timeout
            }
            response = {
                'type': 'command_response',
                'command': command,
                'result': stats,
                'timestamp': time.time()
            }
            await websocket.send(json.dumps(response))
            
        elif command == 'echo':
            # 回显消息
            response = {
                'type': 'command_response',
                'command': command,
                'result': params.get('message', ''),
                'timestamp': time.time()
            }
            await websocket.send(json.dumps(response))
            
        else:
            # 未知命令
            response = {
                'type': 'command_response',
                'command': command,
                'error': f"未知命令: {command}",
                'timestamp': time.time()
            }
            await websocket.send(json.dumps(response))
            
        logger.info(f"执行命令: {command}")
        
    async def handle_file_message(self, websocket: WebSocketServerProtocol, data: dict):
        """处理文件相关消息"""
        file_action = data.get('action', '')
        
        if file_action == 'upload_request':
            # 处理文件上传请求
            file_info = data.get('file_info', {})
            response = {
                'type': 'file_response',
                'action': 'upload_request',
                'status': 'accepted',
                'file_id': f"file_{int(time.time())}",
                'timestamp': time.time()
            }
            await websocket.send(json.dumps(response))
            
        elif file_action == 'download_request':
            # 处理文件下载请求
            file_id = data.get('file_id', '')
            response = {
                'type': 'file_response',
                'action': 'download_request',
                'status': 'processing',
                'file_id': file_id,
                'timestamp': time.time()
            }
            await websocket.send(json.dumps(response))
            
        logger.info(f"处理文件操作: {file_action}")
        
    async def handle_notification(self, websocket: WebSocketServerProtocol, data: dict):
        """处理通知消息"""
        notification_type = data.get('notification_type', 'info')
        message = data.get('message', '')
        
        # 根据通知类型处理
        if notification_type == 'alert':
            # 紧急通知，广播给所有用户
            await self.broadcast_message({
                'type': 'notification',
                'notification_type': 'alert',
                'message': message,
                'timestamp': time.time()
            })
        elif notification_type == 'info':
            # 信息通知，只发送给当前用户
            await websocket.send(json.dumps(data))
            
        logger.info(f"处理通知: {notification_type} - {message}")
        
    async def handle_heartbeat(self, websocket: WebSocketServerProtocol, data: dict):
        """处理心跳消息"""
        if websocket in self.client_info:
            # 更新最后心跳时间
            self.client_info[websocket]['last_heartbeat'] = time.time()
            
            # 发送心跳响应
            response = {
                'type': 'heartbeat_response',
                'timestamp': time.time(),
                'server_time': time.time()
            }
            await websocket.send(json.dumps(response))
            
            logger.debug(f"收到心跳: {self.client_info[websocket]['client_id']}")
            
    async def handle_room_message(self, websocket: WebSocketServerProtocol, data: dict):
        """处理房间相关消息"""
        room_action = data.get('action', '')
        room_id = data.get('room_id', '')
        
        if room_action == 'join_room':
            # 加入房间
            if 'rooms' not in self.client_info[websocket]:
                self.client_info[websocket]['rooms'] = set()
            self.client_info[websocket]['rooms'].add(room_id)
            
            response = {
                'type': 'room_response',
                'action': 'join_room',
                'room_id': room_id,
                'status': 'success',
                'timestamp': time.time()
            }
            await websocket.send(json.dumps(response))
            
        elif room_action == 'leave_room':
            # 离开房间
            if 'rooms' in self.client_info[websocket]:
                self.client_info[websocket]['rooms'].discard(room_id)
                
            response = {
                'type': 'room_response',
                'action': 'leave_room',
                'room_id': room_id,
                'status': 'success',
                'timestamp': time.time()
            }
            await websocket.send(json.dumps(response))
            
        logger.info(f"房间操作: {room_action} - {room_id}")
        
    async def handle_user_message(self, websocket: WebSocketServerProtocol, data: dict):
        """处理用户相关消息"""
        user_action = data.get('action', '')
        
        if user_action == 'update_profile':
            # 更新用户资料
            profile = data.get('profile', {})
            if websocket in self.client_info:
                self.client_info[websocket]['profile'] = profile
                
            response = {
                'type': 'user_response',
                'action': 'update_profile',
                'status': 'success',
                'timestamp': time.time()
            }
            await websocket.send(json.dumps(response))
            
        elif user_action == 'get_profile':
            # 获取用户资料
            profile = self.client_info[websocket].get('profile', {})
            response = {
                'type': 'user_response',
                'action': 'get_profile',
                'profile': profile,
                'timestamp': time.time()
            }
            await websocket.send(json.dumps(response))
            
        logger.info(f"用户操作: {user_action}")
        
    async def handle_arbitrary_message(self, websocket: WebSocketServerProtocol, data: dict):
        """处理任意其他类型的消息"""
        # 记录未知消息类型
        message_type = data.get('type', 'unknown')
        logger.info(f"处理任意消息类型: {message_type}")
        
        # 可以选择是否广播给其他客户端
        if data.get('broadcast', False):
            await self.broadcast_message(data, exclude_websocket=websocket)
            logger.info(f"广播任意消息: {message_type}")
        
        # 发送确认响应
        response = {
            'type': 'message_received',
            'original_type': message_type,
            'status': 'processed',
            'timestamp': time.time()
        }
        await websocket.send(json.dumps(response))
            
    async def broadcast_message(self, message: dict, exclude_websocket: Optional[WebSocketServerProtocol] = None):
        """广播消息给所有客户端"""
        if not self.clients:
            return
            
        message_str = json.dumps(message)
        broadcast_tasks = []
        
        for websocket in self.clients:
            if websocket != exclude_websocket and websocket.open:
                broadcast_tasks.append(websocket.send(message_str))
                
        if broadcast_tasks:
            await asyncio.gather(*broadcast_tasks, return_exceptions=True)
            
    async def handle_client(self, websocket: WebSocketServerProtocol, path: str):
        """处理单个客户端连接"""
        await self.register_client(websocket)
        
        try:
            # 发送欢迎消息
            welcome_message = {
                'type': 'system',
                'message': '欢迎连接到WebSocket服务器！',
                'timestamp': time.time(),
                'client_id': self.client_info[websocket]['client_id']
            }
            await websocket.send(json.dumps(welcome_message))
            
            # 处理客户端消息
            async for message in websocket:
                await self.handle_message(websocket, message)
                
        except websockets.exceptions.ConnectionClosed:
            logger.info("客户端连接关闭")
        except Exception as e:
            logger.error(f"处理客户端时出错: {e}")
        finally:
            await self.unregister_client(websocket)
            
    async def start_server(self):
        """启动WebSocket服务器"""
        # 启动ping任务
        ping_task = asyncio.create_task(self.ping_clients())
        
        try:
            # 启动WebSocket服务器
            server = await websockets.serve(
                self.handle_client,
                self.host,
                self.port,
                ping_interval=None,  # 禁用内置ping，使用自定义实现
                ping_timeout=None
            )
            
            logger.info(f"WebSocket服务器启动成功，监听地址: ws://{self.host}:{self.port}")
            logger.info(f"Ping间隔: {self.ping_interval}秒, Pong超时: {self.pong_timeout}秒")
            
            # 保持服务器运行
            await server.wait_closed()
            
        except Exception as e:
            logger.error(f"启动服务器失败: {e}")
        finally:
            ping_task.cancel()
            
    def run(self):
        """运行服务器（同步接口）"""
        try:
            asyncio.run(self.start_server())
        except KeyboardInterrupt:
            logger.info("服务器被用户中断")
        except Exception as e:
            logger.error(f"服务器运行出错: {e}")

def main():
    """主函数"""
    # 创建并启动服务器
    server = WebSocketServer(host="0.0.0.0", port=18080)
    
    print("=" * 50)
    print("WebSocket服务器启动中...")
    print(f"监听地址: ws://0.0.0.0:18080")
    print("按 Ctrl+C 停止服务器")
    print("=" * 50)
    
    server.run()

if __name__ == "__main__":
    main()
