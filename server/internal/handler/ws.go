// server/internal/handler/ws.go
package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"story-maker/server/config"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // 生产环境需严格校验 Origin
	},
}

// WSMessage WebSocket 消息结构
type WSMessage struct {
	Type string      `json:"type"` // task_update, system_notification
	Data interface{} `json:"data"`
}

// Client WebSocket 客户端连接
type Client struct {
	userID uint
	conn   *websocket.Conn
	send   chan []byte
	hub    *Hub
}

// Hub WebSocket 连接管理中心
type Hub struct {
	clients    map[uint]map[*Client]bool // userID -> clients
	register   chan *Client
	unregister chan *Client
	broadcast  chan *BroadcastMessage
	mu         sync.RWMutex
}

// BroadcastMessage 广播消息
type BroadcastMessage struct {
	UserID  uint
	Message []byte
}

// NewHub 创建 Hub 实例
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[uint]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *BroadcastMessage, 256),
	}
}

// Run 启动 Hub 主循环
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if h.clients[client.userID] == nil {
				h.clients[client.userID] = make(map[*Client]bool)
			}
			h.clients[client.userID][client] = true
			h.mu.Unlock()
			log.Printf("WebSocket client registered: user_id=%d", client.userID)

		case client := <-h.unregister:
			h.mu.Lock()
			if clients, ok := h.clients[client.userID]; ok {
				if _, ok := clients[client]; ok {
					delete(clients, client)
					close(client.send)
					if len(clients) == 0 {
						delete(h.clients, client.userID)
					}
				}
			}
			h.mu.Unlock()
			log.Printf("WebSocket client unregistered: user_id=%d", client.userID)

		case message := <-h.broadcast:
			h.mu.RLock()
			clients := h.clients[message.UserID]
			h.mu.RUnlock()

			for client := range clients {
				select {
				case client.send <- message.Message:
				default:
					close(client.send)
					h.mu.Lock()
					delete(h.clients[message.UserID], client)
					h.mu.Unlock()
				}
			}
		}
	}
}

// NotifyUser 向指定用户推送消息
func (h *Hub) NotifyUser(userID uint, message interface{}) error {
	wsMsg := WSMessage{
		Type: "task_update",
		Data: message,
	}

	msgBytes, err := json.Marshal(wsMsg)
	if err != nil {
		return err
	}

	h.broadcast <- &BroadcastMessage{
		UserID:  userID,
		Message: msgBytes,
	}

	return nil
}

// NotifyUserWithType 向指定用户推送指定类型的消息
func (h *Hub) NotifyUserWithType(userID uint, msgType string, message interface{}) error {
	wsMsg := WSMessage{
		Type: msgType,
		Data: message,
	}

	msgBytes, err := json.Marshal(wsMsg)
	if err != nil {
		return err
	}

	h.broadcast <- &BroadcastMessage{
		UserID:  userID,
		Message: msgBytes,
	}

	return nil
}

// readPump 读取客户端消息（心跳检测）
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket read error: %v", err)
			}
			break
		}
	}
}

// writePump 向客户端写入消息
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// 每条消息独立发送一个 WebSocket 帧，避免前端 JSON.parse 只能解析第一条
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

			// 逐条发送队列中积压的消息
			n := len(c.send)
			for i := 0; i < n; i++ {
				if err := c.conn.WriteMessage(websocket.TextMessage, <-c.send); err != nil {
					return
				}
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// WSHandler WebSocket 连接处理器
type WSHandler struct {
	hub *Hub
}

// NewWSHandler 创建 WSHandler 实例
func NewWSHandler(hub *Hub) *WSHandler {
	return &WSHandler{hub: hub}
}

// HandleWebSocket 处理 WebSocket 连接请求
func (h *WSHandler) HandleWebSocket(c *gin.Context) {
	// 从 query param 获取 token 并解析 user_id
	tokenStr := c.Query("token")
	if tokenStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
		return
	}

	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.Global.JWT.Secret), nil
	})
	if err != nil || !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid claims"})
		return
	}

	userIDFloat, ok := claims["user_id"].(float64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user_id in token"})
		return
	}
	userID := uint(userIDFloat)

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	client := &Client{
		userID: userID,
		conn:   conn,
		send:   make(chan []byte, 256),
		hub:    h.hub,
	}

	h.hub.register <- client

	go client.writePump()
	go client.readPump()
}
