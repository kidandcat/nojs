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
	
	// Main streaming chat page
	server.Route("/", chatStreamHandler)
	
	// Message send handler (returns minimal response)
	server.Route("/send", sendMessageHandler)

	fmt.Println("🚀 Global Chat Demo running on http://localhost:8080")
	log.Fatal(server.Start(":8080"))
}

// Main chat page with HTML streaming
func chatStreamHandler(ctx *nojs.Context) error {
	// Check if this is just a form submission response
	if ctx.Query("sent") == "1" {
		// Return a minimal page that redirects back to streaming
		return ctx.HTML(http.StatusOK, h.HTML(
			h.Head(
				h.Meta(g.Attr("http-equiv", "refresh"), g.Attr("content", "0;url=/")),
			),
			h.Body(g.Text("Message sent...")),
		))
	}

	// Enable streaming
	stream, err := ctx.Stream()
	if err != nil {
		return fallbackHandler(ctx)
	}

	// Get username from cookie
	username := ""
	if cookie, err := ctx.Request.Cookie("chat_username"); err == nil {
		username = cookie.Value
	}

	// Start streaming the page
	err = stream.StreamPage("Global Chat - NoJS Demo", []string{"/static/style.css"})
	if err != nil {
		return err
	}

	// Write the main container using gomponents
	err = stream.WriteNode(
		h.Div(h.Class("chat-container"),
			h.Div(h.Class("chat-header"),
				h.H1(g.Text("🌍 Global Chat")),
				h.P(g.Text("Chat with anyone, anywhere! (Streaming)")),
			),
			h.Div(h.Class("chat-wrapper"),
				h.Div(h.ID("chat-messages"), h.Class("chat-messages")),
			),
		),
	)
	if err != nil {
		return err
	}

	// Track how many messages we've sent
	messagesSent := 0

	// Send existing messages
	mu.RLock()
	for _, msg := range messages {
		err = stream.WriteNode(renderStreamingMessage(msg, messagesSent))
		if err != nil {
			mu.RUnlock()
			return err
		}
		messagesSent++
	}
	initialCount := len(messages)
	mu.RUnlock()

	// Add the form at the bottom using CSS positioning
	err = stream.WriteNode(
		h.Form(
			h.Class("message-form"),
			h.Action("/send"),
			h.Method("POST"),
			h.Style("position: fixed; bottom: 0; left: 0; right: 0; background: var(--bg-secondary);"),
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
		// Add padding to chat-messages for the form
		g.Raw(`<style>
			#chat-messages {
				padding-bottom: 100px;
			}
		</style>`),
	)
	if err != nil {
		return err
	}

	// Keep streaming new messages
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Request.Context().Done():
			return stream.EndHTML()
		case <-ticker.C:
			// Check for new messages
			mu.RLock()
			currentCount := len(messages)
			if currentCount > initialCount {
				// Send new messages
				for i := initialCount; i < currentCount; i++ {
					err = stream.WriteNode(renderStreamingMessage(messages[i], messagesSent))
					if err != nil {
						mu.RUnlock()
						return err
					}
					messagesSent++
				}
				initialCount = currentCount
			}
			mu.RUnlock()
		}
	}
}

// Render a message with CSS to position it correctly in the stream
func renderStreamingMessage(msg Message, index int) g.Node {
	return g.Group([]g.Node{
		// The message div
		h.Div(
			h.Class("message"),
			h.ID("msg-"+msg.ID),
			h.Style(fmt.Sprintf("position: absolute; top: %dpx; left: 30px; right: 30px;", 120+(index*80))),
			h.Div(h.Class("message-header"),
				h.Span(h.Class("username"), h.Style("color: "+msg.Color), g.Text(msg.Username)),
				h.Span(h.Class("timestamp"), g.Text(msg.Timestamp.Format("15:04:05"))),
			),
			h.Div(h.Class("message-text"), g.Text(msg.Text)),
		),
		// Update the height of the messages container
		g.Raw(fmt.Sprintf(`<style>
			#chat-messages {
				min-height: %dpx;
				position: relative;
			}
		</style>`, 200+(index*80))),
	})
}

// Fallback handler for non-streaming browsers
func fallbackHandler(ctx *nojs.Context) error {
	username := ""
	if cookie, err := ctx.Request.Cookie("chat_username"); err == nil {
		username = cookie.Value
	}

	mu.RLock()
	messagesCopy := make([]Message, len(messages))
	copy(messagesCopy, messages)
	mu.RUnlock()

	messageNodes := []g.Node{}
	for _, msg := range messagesCopy {
		messageNodes = append(messageNodes, h.Div(
			h.Class("message"),
			h.ID("msg-"+msg.ID),
			h.Div(h.Class("message-header"),
				h.Span(h.Class("username"), h.Style("color: "+msg.Color), g.Text(msg.Username)),
				h.Span(h.Class("timestamp"), g.Text(msg.Timestamp.Format("15:04:05"))),
			),
			h.Div(h.Class("message-text"), g.Text(msg.Text)),
		))
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
	
	// Redirect back with a flag
	return ctx.Redirect(http.StatusSeeOther, "/?sent=1")
}