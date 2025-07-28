package chat

import (
	"fmt"
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

type ChatDemo struct {
	messages      []Message
	mu            sync.RWMutex
	colors        []string
	colorMap      map[string]string
	colorMu       sync.Mutex
	userHashes    map[string]string
	hashMu        sync.Mutex
	streamClients map[chan Message]bool
	streamMu      sync.Mutex
	prefix        string
}

func NewChatDemo() *ChatDemo {
	return &ChatDemo{
		messages:      []Message{},
		colors:        []string{"#FF6B6B", "#4ECDC4", "#45B7D1", "#F7B731", "#5F27CD", "#00D2D3", "#FF9FF3", "#54A0FF"},
		colorMap:      make(map[string]string),
		userHashes:    make(map[string]string),
		streamClients: make(map[chan Message]bool),
	}
}

func (c *ChatDemo) getUserColor(username string) string {
	c.colorMu.Lock()
	defer c.colorMu.Unlock()
	
	if color, exists := c.colorMap[username]; exists {
		return color
	}
	
	color := c.colors[len(c.colorMap)%len(c.colors)]
	c.colorMap[username] = color
	return color
}

func (c *ChatDemo) getUserHash(username string) string {
	c.hashMu.Lock()
	defer c.hashMu.Unlock()
	
	if hash, exists := c.userHashes[username]; exists {
		return hash
	}
	
	hash := fmt.Sprintf("%04d", time.Now().UnixNano()%10000)
	c.userHashes[username] = hash
	return hash
}

func (c *ChatDemo) broadcastMessage(msg Message) {
	c.streamMu.Lock()
	defer c.streamMu.Unlock()
	
	for client := range c.streamClients {
		select {
		case client <- msg:
		default:
		}
	}
}

func (c *ChatDemo) RegisterRoutes(server *nojs.Server, prefix string) {
	c.prefix = prefix
	server.Route(prefix, c.chatPageHandler)
	server.Route(prefix+"/messages", c.messagesStreamHandler)
	server.Route(prefix+"/send", c.sendMessageHandler)
}

func (c *ChatDemo) chatPageHandler(ctx *nojs.Context) error {
	username := ""
	if cookie, err := ctx.Request.Cookie("chat_username"); err == nil {
		username = cookie.Value
	}

	page := nojs.Page{
		Title: "Global Chat - NoJS Demo",
		CSS:   []string{"/static/style.css"},
		Body: h.Div(h.Class("chat-container"),
			h.Div(h.Class("chat-header"),
				h.A(h.Href("/"), h.Class("back-button"), 
					g.Raw(`<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<polyline points="15 18 9 12 15 6"></polyline>
					</svg>`),
					g.Text("Back to NoJS"),
				),
				h.H1(g.Text("🌍 Global Chat")),
				h.P(g.Text("Chat with anyone, anywhere!")),
			),
			h.Div(h.Class("chat-wrapper"),
				h.IFrame(
					h.Src(c.prefix+"/messages"),
					h.Class("chat-messages"),
					h.Style("width: 100%; flex: 1; border: none;"),
				),
				nojs.Form(
					nojs.FormConfig{
						Action: c.prefix+"/send",
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

func (c *ChatDemo) messagesStreamHandler(ctx *nojs.Context) error {
	stream, err := ctx.Stream()
	if err != nil {
		return c.messagesStaticHandler(ctx)
	}

	err = stream.StreamPage("Messages", nil)
	if err != nil {
		return err
	}

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

	msgChan := make(chan Message, 10)
	c.streamMu.Lock()
	c.streamClients[msgChan] = true
	c.streamMu.Unlock()

	defer func() {
		c.streamMu.Lock()
		delete(c.streamClients, msgChan)
		c.streamMu.Unlock()
		close(msgChan)
	}()

	c.mu.RLock()
	for _, msg := range c.messages {
		err = stream.WriteNode(c.renderMessage(msg))
		if err != nil {
			c.mu.RUnlock()
			return err
		}
	}
	c.mu.RUnlock()

	for {
		select {
		case <-ctx.Request.Context().Done():
			return stream.EndHTML()
		case msg := <-msgChan:
			err = stream.WriteNode(c.renderMessage(msg))
			if err != nil {
				return err
			}
		case <-time.After(30 * time.Second):
			err = stream.KeepAlive()
			if err != nil {
				return err
			}
		}
	}
}

func (c *ChatDemo) messagesStaticHandler(ctx *nojs.Context) error {
	c.mu.RLock()
	messagesCopy := make([]Message, len(c.messages))
	copy(messagesCopy, c.messages)
	c.mu.RUnlock()

	messageNodes := []g.Node{}
	for _, msg := range messagesCopy {
		messageNodes = append(messageNodes, c.renderMessage(msg))
	}

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

func (c *ChatDemo) renderMessage(msg Message) g.Node {
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

func (c *ChatDemo) sendMessageHandler(ctx *nojs.Context) error {
	if ctx.Request.Method != "POST" {
		return ctx.Redirect(http.StatusSeeOther, c.prefix)
	}
	
	username := ctx.Request.FormValue("username")
	text := ctx.Request.FormValue("text")
	
	if username == "" || text == "" {
		return ctx.Redirect(http.StatusSeeOther, c.prefix)
	}
	
	sessionID := ""
	if cookie, err := ctx.Request.Cookie("chat_session"); err == nil {
		sessionID = cookie.Value
	} else {
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
	
	http.SetCookie(ctx.ResponseWriter, &http.Cookie{
		Name:     "chat_username",
		Value:    username,
		Path:     "/",
		MaxAge:   30 * 24 * 60 * 60,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	
	userKey := username + ":" + sessionID
	userHash := c.getUserHash(userKey)
	
	c.mu.Lock()
	msg := Message{
		ID:        strconv.FormatInt(time.Now().UnixNano(), 10),
		Username:  username,
		UserHash:  userHash,
		Text:      text,
		Timestamp: time.Now(),
		Color:     c.getUserColor(userKey),
	}
	c.messages = append(c.messages, msg)
	c.mu.Unlock()
	
	go c.broadcastMessage(msg)
	
	return ctx.Redirect(http.StatusSeeOther, c.prefix)
}