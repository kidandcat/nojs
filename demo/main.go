package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
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
	sseClients = make(map[chan string]bool)
	sseMu     sync.Mutex
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

func broadcastMessage(msg Message) {
	sseMu.Lock()
	defer sseMu.Unlock()
	
	msgHTML := fmt.Sprintf(`
		<div class="message" id="msg-%s">
			<div class="message-header">
				<span class="username" style="color: %s">%s</span>
				<span class="timestamp">%s</span>
			</div>
			<div class="message-text">%s</div>
		</div>
	`, msg.ID, msg.Color, msg.Username, msg.Timestamp.Format("15:04:05"), msg.Text)
	
	data := fmt.Sprintf("data: %s\n\n", strings.ReplaceAll(msgHTML, "\n", " "))
	
	for client := range sseClients {
		select {
		case client <- data:
		default:
			close(client)
			delete(sseClients, client)
		}
	}
}

func main() {
	server := nojs.NewServer()
	
	// Serve static files
	server.Static("/static/", "./static")
	
	// Main chat page
	server.Route("/", chatHandler)
	
	// Message form handler
	server.Route("/send", sendMessageHandler)
	
	// SSE endpoint
	server.Route("/events", sseHandler)

	fmt.Println("🚀 Global Chat Demo running on http://localhost:8080")
	log.Fatal(server.Start(":8080"))
}

func chatHandler(ctx *nojs.Context) error {
	mu.RLock()
	messagesCopy := make([]Message, len(messages))
	copy(messagesCopy, messages)
	mu.RUnlock()

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
				renderMessageForm("", ""),
			),
		),
		Scripts: []g.Node{
			g.Raw(`<script>
				// SSE for real-time updates
				const eventSource = new EventSource('/events');
				eventSource.onmessage = function(event) {
					const chatMessages = document.getElementById('chat-messages');
					chatMessages.insertAdjacentHTML('beforeend', event.data);
					chatMessages.scrollTop = chatMessages.scrollHeight;
				};
				
				// Auto-scroll to bottom on page load
				document.getElementById('chat-messages').scrollTop = document.getElementById('chat-messages').scrollHeight;
			</script>`),
		},
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
		// Re-render the page with error
		return chatHandler(ctx)
	}
	
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
	
	// Broadcast to SSE clients
	go broadcastMessage(msg)
	
	// Redirect back to chat
	return ctx.Redirect(http.StatusSeeOther, "/")
}

func sseHandler(ctx *nojs.Context) error {
	// Set SSE headers
	ctx.ResponseWriter.Header().Set("Content-Type", "text/event-stream")
	ctx.ResponseWriter.Header().Set("Cache-Control", "no-cache")
	ctx.ResponseWriter.Header().Set("Connection", "keep-alive")
	
	// Create client channel
	client := make(chan string)
	
	// Register client
	sseMu.Lock()
	sseClients[client] = true
	sseMu.Unlock()
	
	// Remove client on disconnect
	defer func() {
		sseMu.Lock()
		delete(sseClients, client)
		sseMu.Unlock()
		close(client)
	}()
	
	// Send events to client
	for {
		select {
		case msg := <-client:
			fmt.Fprint(ctx.ResponseWriter, msg)
			ctx.ResponseWriter.(interface{ Flush() }).Flush()
		case <-ctx.Request.Context().Done():
			return nil
		}
	}
}