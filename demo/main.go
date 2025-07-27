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
	// Channels to notify about new messages to streaming clients
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
	// Get username from cookie
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
				// Use iframe for streaming messages
				h.IFrame(
					h.Src("/messages"),
					h.Class("chat-messages"),
					h.ID("chat-messages"),
					h.Style("width: 100%; flex: 1; border: none; display: block;"),
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
	err = stream.StreamPage("Messages", []string{"/static/style.css"})
	if err != nil {
		return err
	}

	// Add custom styles for iframe content with more specific selectors
	err = stream.WriteNode(
		g.Raw(`<style>
			/* Reset and base styles */
			* {
				box-sizing: border-box;
			}
			
			html, body {
				margin: 0;
				padding: 0;
				background: transparent;
				color: #e4e6eb;
				font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
			}
			
			body {
				padding: 20px;
				overflow-y: auto;
			}
			
			/* Messages wrapper */
			.messages-wrapper {
				display: block !important;
				width: 100% !important;
			}
			
			/* Message container - force block display */
			.messages-wrapper > .message,
			body > div.message,
			.message {
				display: block !important;
				width: calc(100% - 40px) !important;
				max-width: 100% !important;
				background: #1e2541 !important;
				padding: 16px 20px !important;
				border-radius: 12px !important;
				margin: 0 0 15px 0 !important;
				border: 1px solid rgba(255, 255, 255, 0.1) !important;
				transition: all 0.2s ease !important;
				clear: both !important;
				float: none !important;
				position: relative !important;
				box-sizing: border-box !important;
			}
			
			.message:hover {
				background: #252b49 !important;
				transform: translateX(4px);
				box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
			}
			
			.message-header {
				display: flex !important;
				justify-content: space-between !important;
				align-items: center !important;
				margin-bottom: 8px !important;
				width: 100% !important;
			}
			
			.username {
				font-weight: 600 !important;
				font-size: 1.1em !important;
				text-shadow: 0 1px 3px rgba(0, 0, 0, 0.3) !important;
				display: inline-block !important;
			}
			
			.timestamp {
				font-size: 0.85em !important;
				color: #b0b3b8 !important;
				opacity: 0.7 !important;
				display: inline-block !important;
			}
			
			.message-text {
				color: #e4e6eb !important;
				line-height: 1.5 !important;
				word-wrap: break-word !important;
				display: block !important;
				width: 100% !important;
			}
			
			/* Ensure new messages appear with animation */
			@keyframes slideIn {
				from {
					opacity: 0;
					transform: translateY(10px);
				}
				to {
					opacity: 1;
					transform: translateY(0);
				}
			}
			
			.message {
				animation: slideIn 0.3s ease-out;
			}
		</style>`),
	)
	if err != nil {
		return err
	}

	// Create a channel for this client
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

	// Create a wrapper div to contain all messages
	err = stream.WriteNode(g.Raw(`<div class="messages-wrapper">`))
	if err != nil {
		return err
	}

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

	page := nojs.Page{
		Title: "Messages",
		CSS:   []string{"/static/style.css"},
		Body: h.Body(
			g.Raw(`<style>
				* {
					box-sizing: border-box;
				}
				html, body {
					margin: 0;
					padding: 0;
					background: transparent;
					color: #e4e6eb;
					font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
				}
				body {
					padding: 20px;
				}
				.message {
					display: block !important;
					width: 100% !important;
					background: #1e2541 !important;
					padding: 16px 20px !important;
					border-radius: 12px !important;
					margin-bottom: 15px !important;
					border: 1px solid rgba(255, 255, 255, 0.1) !important;
					clear: both !important;
					float: none !important;
				}
				.message-header {
					display: flex !important;
					justify-content: space-between !important;
					align-items: center !important;
					margin-bottom: 8px !important;
				}
				.username {
					font-weight: 600 !important;
					font-size: 1.1em !important;
					text-shadow: 0 1px 3px rgba(0, 0, 0, 0.3) !important;
				}
				.timestamp {
					font-size: 0.85em !important;
					color: #b0b3b8 !important;
					opacity: 0.7 !important;
				}
				.message-text {
					color: #e4e6eb !important;
					line-height: 1.5 !important;
					word-wrap: break-word !important;
					display: block !important;
				}
			</style>`),
			g.Group(messageNodes),
		),
	}

	return ctx.HTML(http.StatusOK, page.Render())
}

// Render a single message
func renderMessage(msg Message) g.Node {
	return h.Div(
		h.Class("message"),
		h.ID("msg-"+msg.ID),
		h.Div(h.Class("message-header"),
			h.Span(h.Class("username"), h.Style("color: "+msg.Color), g.Text(msg.Username)),
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
	
	// Set username cookie
	http.SetCookie(ctx.ResponseWriter, &http.Cookie{
		Name:     "chat_username",
		Value:    username,
		Path:     "/",
		MaxAge:   30 * 24 * 60 * 60,
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
	
	// Broadcast to all streaming clients
	go broadcastMessage(msg)
	
	// Redirect back to chat
	return ctx.Redirect(http.StatusSeeOther, "/")
}