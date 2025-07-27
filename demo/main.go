package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/kidandcat/nojs"
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

type Message struct {
	ID        string
	Username  string
	UserHash  string
	Text      string
	Timestamp time.Time
	Color     string
}

var (
	messages []Message
	mu       sync.RWMutex
	colors   = []string{"#FF6B6B", "#4ECDC4", "#45B7D1", "#F7B731", "#5F27CD", "#00D2D3", "#FF9FF3", "#54A0FF"}
	colorMap = make(map[string]string)
	colorMu  sync.Mutex
	// User hashes for Discord-like identification
	userHashes = make(map[string]string)
	hashMu     sync.Mutex
	// Channels to notify streaming clients about new messages
	streamClients = make(map[chan Message]bool)
	streamMu      sync.Mutex
)

func getUserColor(username string) string {
	colorMu.Lock()
	defer colorMu.Unlock()
	
	if color, exists := colorMap[username]; exists {
		return color
	}
	
	color := colors[len(colorMap)%len(colors)]
	colorMap[username] = color
	return color
}

func getUserHash(username string) string {
	hashMu.Lock()
	defer hashMu.Unlock()
	
	// Use username + cookie value as key for consistent hash per session
	if hash, exists := userHashes[username]; exists {
		return hash
	}
	
	// Generate a random 4-digit hash
	hash := fmt.Sprintf("%04d", time.Now().UnixNano()%10000)
	userHashes[username] = hash
	return hash
}

func broadcastMessage(msg Message) {
	streamMu.Lock()
	defer streamMu.Unlock()
	
	for client := range streamClients {
		select {
		case client <- msg:
		default:
			// Client not ready, skip
		}
	}
}

func main() {
	server := nojs.NewServer()
	
	// Serve static files
	server.Static("/static/", "./static")
	
	// Main chat page
	server.Route("/", chatPageHandler)
	
	// Streaming messages endpoint (for iframe)
	server.Route("/messages", messagesStreamHandler)
	
	// Message send handler
	server.Route("/send", sendMessageHandler)

	fmt.Println("🚀 Global Chat Demo running on http://localhost:8080")
	log.Fatal(server.Start(":8080"))
}

// Main chat page with iframe for messages
func chatPageHandler(ctx *nojs.Context) error {
	username := ""
	if cookie, err := ctx.Request.Cookie("chat_username"); err == nil {
		username = cookie.Value
	}

	page := nojs.Page{
		Title: "Global Chat - NoJS Demo",
		CSS:   []string{"/static/style.css"},
		Body: h.Div(h.Class("chat-container"),
			h.Div(h.Class("chat-header"),
				h.H1(g.Text("🌍 Global Chat")),
				h.P(g.Text("Chat with anyone, anywhere!")),
			),
			h.Div(h.Class("chat-wrapper"),
				// iframe for streaming messages
				h.IFrame(
					h.Src("/messages"),
					h.Class("chat-messages"),
					h.Style("width: 100%; flex: 1; border: none;"),
				),
				// Message form
				nojs.Form(
					nojs.FormConfig{
						Action: "/send",
						Method: "POST",
						Class:  "message-form",
					},
					h.Div(h.Class("form-group"),
						h.Input(
							h.Type("text"),
							h.Name("username"),
							h.Placeholder("Your name"),
							h.Required(),
							h.Class("username-input"),
							g.If(username != "", h.Value(username)),
						),
						h.Input(
							h.Type("text"),
							h.Name("text"),
							h.Placeholder("Type a message..."),
							h.Required(),
							h.Class("message-input"),
							h.AutoFocus(),
						),
						h.Button(
							h.Type("submit"),
							h.Class("send-button"),
							h.Span(g.Text("Send")),
							g.Raw(`<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
								<line x1="22" y1="2" x2="11" y2="13"></line>
								<polygon points="22 2 15 22 11 13 2 9 22 2"></polygon>
							</svg>`),
						),
					),
				),
			),
		),
	}

	return ctx.HTML(http.StatusOK, page.Render())
}

// Stream messages in the iframe
func messagesStreamHandler(ctx *nojs.Context) error {
	// Enable streaming
	stream, err := ctx.Stream()
	if err != nil {
		return messagesStaticHandler(ctx)
	}

	// Start HTML document
	err = stream.StreamPage("Messages", nil)
	if err != nil {
		return err
	}

	// Simple styles for messages
	err = stream.WriteNode(
		nojs.Styles(map[string]map[string]string{
			"body": {
				"margin": "0",
				"padding": "20px",
				"background": "transparent",
				"font-family": "system-ui, sans-serif",
				"color": "#e4e6eb",
			},
			".message": {
				"background": "#1e2541",
				"padding": "16px 20px",
				"border-radius": "12px",
				"margin-bottom": "15px",
				"border": "1px solid rgba(255, 255, 255, 0.1)",
			},
			".message-header": {
				"display": "flex",
				"justify-content": "space-between",
				"align-items": "center",
				"margin-bottom": "8px",
			},
			".username-wrapper": {
				"display": "flex",
				"align-items": "baseline",
				"gap": "6px",
			},
			".username": {
				"font-weight": "600",
				"font-size": "1.1em",
			},
			".user-hash": {
				"color": "#6a6d72",
				"font-size": "0.9em",
				"font-weight": "400",
			},
			".timestamp": {
				"font-size": "0.85em",
				"color": "#b0b3b8",
				"opacity": "0.7",
			},
			".message-text": {
				"color": "#e4e6eb",
				"line-height": "1.5",
			},
		}),
	)
	if err != nil {
		return err
	}

	// Create channel for this client
	msgChan := make(chan Message, 10)
	streamMu.Lock()
	streamClients[msgChan] = true
	streamMu.Unlock()

	// Clean up on disconnect
	defer func() {
		streamMu.Lock()
		delete(streamClients, msgChan)
		streamMu.Unlock()
		close(msgChan)
	}()

	// Send existing messages
	mu.RLock()
	for _, msg := range messages {
		err = stream.WriteNode(renderMessage(msg))
		if err != nil {
			mu.RUnlock()
			return err
		}
	}
	mu.RUnlock()

	// Keep connection open and stream new messages
	for {
		select {
		case <-ctx.Request.Context().Done():
			return stream.EndHTML()
		case msg := <-msgChan:
			// Stream the new message
			err = stream.WriteNode(renderMessage(msg))
			if err != nil {
				return err
			}
		case <-time.After(30 * time.Second):
			// Send keep-alive
			err = stream.KeepAlive()
			if err != nil {
				return err
			}
		}
	}
}

// Static fallback for non-streaming browsers
func messagesStaticHandler(ctx *nojs.Context) error {
	mu.RLock()
	messagesCopy := make([]Message, len(messages))
	copy(messagesCopy, messages)
	mu.RUnlock()

	messageNodes := []g.Node{}
	for _, msg := range messagesCopy {
		messageNodes = append(messageNodes, renderMessage(msg))
	}

	// Simple page with messages
	return ctx.HTML(http.StatusOK, h.HTML(
		h.Head(
			h.Meta(h.Charset("utf-8")),
			h.Title("Messages"),
			nojs.Style(`
				body {
					margin: 0;
					padding: 20px;
					background: transparent;
					font-family: system-ui, sans-serif;
					color: #e4e6eb;
				}
				.message {
					background: #1e2541;
					padding: 16px 20px;
					border-radius: 12px;
					margin-bottom: 15px;
					border: 1px solid rgba(255, 255, 255, 0.1);
				}
				.message-header {
					display: flex;
					justify-content: space-between;
					align-items: center;
					margin-bottom: 8px;
				}
				.username-wrapper {
					display: flex;
					align-items: baseline;
					gap: 6px;
				}
				.username {
					font-weight: 600;
					font-size: 1.1em;
				}
				.user-hash {
					color: #6a6d72;
					font-size: 0.9em;
					font-weight: 400;
				}
				.timestamp {
					font-size: 0.85em;
					color: #b0b3b8;
					opacity: 0.7;
				}
				.message-text {
					color: #e4e6eb;
					line-height: 1.5;
				}
			`),
		),
		h.Body(messageNodes...),
	))
}

// Render a single message
func renderMessage(msg Message) g.Node {
	return h.Div(
		h.Class("message"),
		h.Div(h.Class("message-header"),
			h.Span(h.Class("username-wrapper"),
				h.Span(h.Class("username"), h.Style("color: "+msg.Color), g.Text(msg.Username)),
				h.Span(h.Class("user-hash"), g.Text("#"+msg.UserHash)),
			),
			h.Span(h.Class("timestamp"), g.Text(msg.Timestamp.Format("15:04:05"))),
		),
		h.Div(h.Class("message-text"), g.Text(msg.Text)),
	)
}

// Handle message sending
func sendMessageHandler(ctx *nojs.Context) error {
	if ctx.Request.Method != "POST" {
		return ctx.Redirect(http.StatusSeeOther, "/")
	}
	
	username := ctx.Request.FormValue("username")
	text := ctx.Request.FormValue("text")
	
	if username == "" || text == "" {
		return ctx.Redirect(http.StatusSeeOther, "/")
	}
	
	// Get or create user session ID from cookie
	sessionID := ""
	if cookie, err := ctx.Request.Cookie("chat_session"); err == nil {
		sessionID = cookie.Value
	} else {
		// Create new session ID
		sessionID = strconv.FormatInt(time.Now().UnixNano(), 36)
		http.SetCookie(ctx.ResponseWriter, &http.Cookie{
			Name:     "chat_session",
			Value:    sessionID,
			Path:     "/",
			MaxAge:   30 * 24 * 60 * 60,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
	}
	
	// Set username cookie
	http.SetCookie(ctx.ResponseWriter, &http.Cookie{
		Name:     "chat_username",
		Value:    username,
		Path:     "/",
		MaxAge:   30 * 24 * 60 * 60,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	
	// Get user hash (unique per username+session combination)
	userKey := username + ":" + sessionID
	userHash := getUserHash(userKey)
	
	// Add message
	mu.Lock()
	msg := Message{
		ID:        strconv.FormatInt(time.Now().UnixNano(), 10),
		Username:  username,
		UserHash:  userHash,
		Text:      text,
		Timestamp: time.Now(),
		Color:     getUserColor(userKey),
	}
	messages = append(messages, msg)
	mu.Unlock()
	
	// Broadcast to all streaming clients
	go broadcastMessage(msg)
	
	// Redirect back to chat
	return ctx.Redirect(http.StatusSeeOther, "/")
}