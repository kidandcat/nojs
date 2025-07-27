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

func main() {
	server := nojs.NewServer()
	
	// Serve static files
	server.Static("/static/", "./static")
	
	// Main chat page with streaming
	server.Route("/", chatStreamHandler)
	
	// Message form handler
	server.Route("/send", sendMessageHandler)

	fmt.Println("🚀 Global Chat Demo running on http://localhost:8080")
	log.Fatal(server.Start(":8080"))
}

func chatStreamHandler(ctx *nojs.Context) error {
	// Enable streaming
	stream, err := ctx.Stream()
	if err != nil {
		// Fallback to non-streaming
		return chatHandler(ctx)
	}

	// Send initial page structure
	err = stream.WriteHTML(`<!DOCTYPE html>
<html>
<head>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<title>Global Chat - NoJS Demo</title>
	<link rel="stylesheet" href="/static/style.css">
</head>
<body>
	<div class="chat-container">
		<div class="chat-header">
			<h1>🌍 Global Chat</h1>
			<p>Chat with anyone, anywhere!</p>
		</div>
		<div class="chat-wrapper">
			<div id="chat-messages" class="chat-messages">
`)
	if err != nil {
		return err
	}

	// Send existing messages
	mu.RLock()
	for _, msg := range messages {
		msgHTML := fmt.Sprintf(`
			<div class="message" id="msg-%s">
				<div class="message-header">
					<span class="username" style="color: %s">%s</span>
					<span class="timestamp">%s</span>
				</div>
				<div class="message-text">%s</div>
			</div>
		`, msg.ID, msg.Color, msg.Username, msg.Timestamp.Format("15:04:05"), msg.Text)
		
		err = stream.WriteHTML(msgHTML)
		if err != nil {
			mu.RUnlock()
			return err
		}
	}
	lastMessageCount := len(messages)
	mu.RUnlock()

	// End messages div but keep connection open
	err = stream.WriteHTML(`
			</div>
`)
	if err != nil {
		return err
	}

	// Get username from cookie
	username := ""
	if cookie, err := ctx.Request.Cookie("chat_username"); err == nil {
		username = cookie.Value
	}

	// Send the form
	formHTML := fmt.Sprintf(`
			<form id="message-form" class="message-form" action="/send" method="POST">
				<div class="form-group">
					<input type="text" name="username" placeholder="Your name" required value="%s" class="username-input">
					<input type="text" name="text" placeholder="Type a message..." required value="" class="message-input" autofocus>
					<button type="submit" class="send-button">
						<span>Send</span>
						<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<line x1="22" y1="2" x2="11" y2="13"></line>
							<polygon points="22 2 15 22 11 13 2 9 22 2"></polygon>
						</svg>
					</button>
				</div>
			</form>
		</div>
	</div>
`, username)
	
	err = stream.WriteHTML(formHTML)
	if err != nil {
		return err
	}

	// Now keep the connection open and stream new messages
	for {
		select {
		case <-ctx.Request.Context().Done():
			// Client disconnected
			return stream.WriteHTML("</body></html>")
		default:
			// Check for new messages
			mu.RLock()
			if len(messages) > lastMessageCount {
				// We have new messages to send
				for i := lastMessageCount; i < len(messages); i++ {
					msg := messages[i]
					
					// Use CSS absolute positioning to place message above the form
					msgHTML := fmt.Sprintf(`
					<div class="message streaming-message" id="msg-%s" style="position: absolute; bottom: %dpx; left: 30px; right: 30px;">
						<div class="message-header">
							<span class="username" style="color: %s">%s</span>
							<span class="timestamp">%s</span>
						</div>
						<div class="message-text">%s</div>
					</div>
					<style>
						#chat-messages { padding-bottom: %dpx !important; }
					</style>
					`, msg.ID, 100 + (i * 80), msg.Color, msg.Username, 
					msg.Timestamp.Format("15:04:05"), msg.Text, 100 + ((i+1) * 80))
					
					err = stream.WriteHTML(msgHTML)
					if err != nil {
						mu.RUnlock()
						return err
					}
				}
				lastMessageCount = len(messages)
			}
			mu.RUnlock()
			
			// Small delay to avoid busy waiting
			stream.Sleep(200 * time.Millisecond)
		}
	}
}

func chatHandler(ctx *nojs.Context) error {
	mu.RLock()
	messagesCopy := make([]Message, len(messages))
	copy(messagesCopy, messages)
	mu.RUnlock()

	// Get username from cookie
	username := ""
	if cookie, err := ctx.Request.Cookie("chat_username"); err == nil {
		username = cookie.Value
	}

	// Create message nodes
	messageNodes := []g.Node{}
	for _, msg := range messagesCopy {
		messageNodes = append(messageNodes, renderMessage(msg))
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
				h.Div(append([]g.Node{h.ID("chat-messages"), h.Class("chat-messages")}, messageNodes...)...),
				renderMessageForm(username, ""),
			),
		),
	}

	return ctx.HTML(http.StatusOK, page.Render())
}

func renderMessage(msg Message) g.Node {
	return h.Div(h.Class("message"), h.ID("msg-"+msg.ID),
		h.Div(h.Class("message-header"),
			h.Span(h.Class("username"), h.Style("color: "+msg.Color), g.Text(msg.Username)),
			h.Span(h.Class("timestamp"), g.Text(msg.Timestamp.Format("15:04:05"))),
		),
		h.Div(h.Class("message-text"), g.Text(msg.Text)),
	)
}

func renderMessageForm(username, errorMsg string) g.Node {
	formNodes := []g.Node{
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
	}
	
	if errorMsg != "" {
		formNodes = append([]g.Node{nojs.Alert(errorMsg, "error")}, formNodes...)
	}
	
	return nojs.Form(
		nojs.FormConfig{
			Action: "/send",
			Method: "POST",
			Class:  "message-form",
		},
		formNodes...,
	)
}

func sendMessageHandler(ctx *nojs.Context) error {
	if ctx.Request.Method != "POST" {
		return ctx.Redirect(http.StatusSeeOther, "/")
	}
	
	username := ctx.Request.FormValue("username")
	text := ctx.Request.FormValue("text")
	
	if username == "" || text == "" {
		// Redirect back with error
		return ctx.Redirect(http.StatusSeeOther, "/")
	}
	
	// Set username cookie (expires in 30 days)
	http.SetCookie(ctx.ResponseWriter, &http.Cookie{
		Name:     "chat_username",
		Value:    username,
		Path:     "/",
		MaxAge:   30 * 24 * 60 * 60, // 30 days
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	
	// Add message
	mu.Lock()
	msg := Message{
		ID:        strconv.FormatInt(time.Now().UnixNano(), 10),
		Username:  username,
		Text:      text,
		Timestamp: time.Now(),
		Color:     getUserColor(username),
	}
	messages = append(messages, msg)
	mu.Unlock()
	
	// Redirect back to chat
	return ctx.Redirect(http.StatusSeeOther, "/")
}