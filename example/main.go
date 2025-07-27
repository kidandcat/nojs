package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/jairo/mavis/nojs"
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// TodoItem represents a todo item
type TodoItem struct {
	ID        int
	Text      string
	Completed bool
	CreatedAt time.Time
}

// ChatMessage represents a chat message
type ChatMessage struct {
	ID        int
	Username  string
	Message   string
	Timestamp time.Time
}

// ChatRoom manages chat messages and subscribers
type ChatRoom struct {
	mu          sync.RWMutex
	messages    []ChatMessage
	subscribers map[string]chan ChatMessage
	nextID      int
}

// In-memory storage for demo
var (
	todos    = make(map[int]*TodoItem)
	nextID   = 1
	chatRoom = &ChatRoom{
		messages:    make([]ChatMessage, 0),
		subscribers: make(map[string]chan ChatMessage),
		nextID:      1,
	}
)

func main() {
	// Initialize with some demo data
	addTodo("Build a web app without JavaScript")
	addTodo("Learn about HTML streaming")
	addTodo("Master server-side rendering")

	// Create server
	server := nojs.NewServer()

	// Add middleware
	server.Use(nojs.Logger())
	server.Use(nojs.Recovery())

	// Routes
	server.Route("/", handleIndex)
	server.Route("/todos", handleTodos)
	server.Route("/todos/add", handleAddTodo)
	server.Route("/todos/toggle", handleToggleTodo)
	server.Route("/todos/delete", handleDeleteTodo)
	server.Route("/chat", handleChat)
	server.Route("/chat/send", handleChatSend)
	server.Route("/chat/stream", handleChatStream)

	// Static files
	server.Static("/static/", "./static")

	log.Println("Server starting on http://localhost:8080")
	log.Fatal(server.Start(":8080"))
}

func handleIndex(ctx *nojs.Context) error {
	content := h.Div(h.Class("container"),
		h.H1(g.Text("NoJS Example Application")),
		h.P(g.Text("This is a demo of the NoJS framework - building web apps without JavaScript!")),
		h.Div(h.Class("grid"),
			nojs.Card("Todo App", h.Div(
				h.P(g.Text("A simple todo application with CRUD operations")),
				h.A(h.Href("/todos"), g.Text("View Todo App →")),
			)),
			nojs.Card("Real-time Chat", h.Div(
				h.P(g.Text("Live chat using HTTP streaming - no WebSockets!")),
				h.A(h.Href("/chat"), g.Text("Join Chat Room →")),
			)),
		),
	)

	page := nojs.Page{
		Title: "NoJS Example",
		CSS:   []string{"/static/style.css"},
		Body:  content,
	}

	return ctx.HTML(200, page.Render())
}

func handleTodos(ctx *nojs.Context) error {
	// Get flash messages
	successFlash := ctx.GetFlash("success")
	errorFlash := ctx.GetFlash("error")

	// Check if we should show the add modal
	showModal := ctx.Query("modal") == "add"

	content := h.Div(h.Class("container"),
		h.H1(g.Text("Todo List")),
		h.P(h.A(h.Href("/"), g.Text("← Back to Home"))),

		// Flash messages
		g.If(successFlash != "", nojs.Alert(successFlash, "success")),
		g.If(errorFlash != "", nojs.Alert(errorFlash, "error")),

		// Add button
		h.Div(h.Class("actions"),
			h.A(h.Href("/todos?modal=add"), h.Class("button"), g.Text("Add New Todo")),
		),

		// Todo list
		renderTodoList(),

		// Add modal
		g.If(showModal, renderAddModal()),

		// Auto-refresh every 10 seconds
		nojs.AutoRefresh(10),
	)

	page := nojs.Page{
		Title: "Todo List - NoJS Example",
		CSS:   []string{"/static/style.css"},
		Body:  content,
	}

	return ctx.HTML(200, page.Render())
}

func handleAddTodo(ctx *nojs.Context) error {
	if ctx.Method() != "POST" {
		return nojs.NewHTTPError(405, "Method not allowed")
	}

	text := ctx.Form("text")
	if text == "" {
		ctx.SetFlash("error", "Todo text is required")
		return ctx.Redirect(303, "/todos?modal=add")
	}

	addTodo(text)
	ctx.SetFlash("success", "Todo added successfully!")
	return ctx.Redirect(303, "/todos")
}

func handleToggleTodo(ctx *nojs.Context) error {
	if ctx.Method() != "POST" {
		return nojs.NewHTTPError(405, "Method not allowed")
	}

	id := 0
	fmt.Sscanf(ctx.Form("id"), "%d", &id)

	if todo, exists := todos[id]; exists {
		todo.Completed = !todo.Completed
		status := "uncompleted"
		if todo.Completed {
			status = "completed"
		}
		ctx.SetFlash("success", fmt.Sprintf("Todo marked as %s", status))
	}

	return ctx.Redirect(303, "/todos")
}

func handleDeleteTodo(ctx *nojs.Context) error {
	if ctx.Method() != "POST" {
		return nojs.NewHTTPError(405, "Method not allowed")
	}

	id := 0
	fmt.Sscanf(ctx.Form("id"), "%d", &id)

	if _, exists := todos[id]; exists {
		delete(todos, id)
		ctx.SetFlash("success", "Todo deleted successfully")
	}

	return ctx.Redirect(303, "/todos")
}

func handleChat(ctx *nojs.Context) error {
	// Check if user wants to change username
	if ctx.Query("change") == "1" {
		// Clear username cookie
		http.SetCookie(ctx.ResponseWriter, &http.Cookie{
			Name:   "chat_username",
			Value:  "",
			Path:   "/",
			MaxAge: -1,
		})
		return ctx.Redirect(303, "/chat")
	}

	// Get username from cookie
	username := ""
	cookie, err := ctx.Request.Cookie("chat_username")
	if err == nil {
		username = cookie.Value
	}

	// Handle username submission
	if ctx.Method() == "POST" && username == "" {
		username = ctx.Form("username")
		if username != "" {
			http.SetCookie(ctx.ResponseWriter, &http.Cookie{
				Name:     "chat_username",
				Value:    username,
				Path:     "/",
				MaxAge:   86400, // 24 hours
				HttpOnly: true,
			})
			return ctx.Redirect(303, "/chat")
		}
	}

	var bodyContent g.Node
	if username == "" {
		bodyContent = renderUsernameForm()
	} else {
		bodyContent = h.Div(
			h.P(g.Text(fmt.Sprintf("Chatting as: %s", username))),
			
			// Chat messages container with iframe for streaming
			h.Div(h.Class("chat-container"),
				h.IFrame(
					h.Src("/chat/stream"),
					h.Class("chat-messages"),
					g.Attr("frameborder", "0"),
				),
			),
			
			// Message input form
			nojs.Form(nojs.FormConfig{
				Action: "/chat/send",
				Method: "POST",
				Class:  "chat-form",
			},
				h.Input(h.Type("hidden"), h.Name("username"), h.Value(username)),
				h.Div(h.Class("chat-input-group"),
					h.Input(
						h.Type("text"),
						h.Name("message"),
						h.Placeholder("Type your message..."),
						h.Required(),
						h.AutoFocus(),
						h.Class("chat-input"),
					),
					h.Button(h.Type("submit"), h.Class("button"), g.Text("Send")),
				),
			),
			
			// Change username link
			h.P(h.Class("change-username"),
				h.A(h.Href("/chat?change=1"), g.Text("Change username")),
			),
		)
	}

	content := h.Div(h.Class("container"),
		h.H1(g.Text("Real-time Chat Room")),
		h.P(h.A(h.Href("/"), g.Text("← Back to Home"))),
		bodyContent,
	)

	page := nojs.Page{
		Title: "Chat Room - NoJS Example",
		CSS:   []string{"/static/style.css"},
		Body:  content,
	}

	return ctx.HTML(200, page.Render())
}

func handleChatSend(ctx *nojs.Context) error {
	if ctx.Method() != "POST" {
		return nojs.NewHTTPError(405, "Method not allowed")
	}

	username := ctx.Form("username")
	message := ctx.Form("message")

	if username == "" || message == "" {
		return ctx.Redirect(303, "/chat")
	}

	// Add message to chat room
	chatRoom.AddMessage(username, message)

	// Set username cookie
	http.SetCookie(ctx.ResponseWriter, &http.Cookie{
		Name:     "chat_username",
		Value:    username,
		Path:     "/",
		MaxAge:   86400, // 24 hours
		HttpOnly: true,
	})

	return ctx.Redirect(303, "/chat")
}

func handleChatStream(ctx *nojs.Context) error {
	stream, err := ctx.Stream()
	if err != nil {
		return err
	}

	// Generate unique subscriber ID
	subscriberID := fmt.Sprintf("sub-%d", time.Now().UnixNano())
	
	// Subscribe to chat updates
	msgChan := chatRoom.Subscribe(subscriberID)
	defer chatRoom.Unsubscribe(subscriberID)

	// We need to render the start manually since we're streaming
	stream.WriteString("<!DOCTYPE html>\n<html>\n<head>\n")
	stream.WriteNode(h.Meta(h.Charset("utf-8")))
	stream.WriteNode(h.Link(h.Rel("stylesheet"), h.Href("/static/style.css")))
	stream.WriteString(`<style>
body { 
	margin: 0; 
	padding: 1rem;
	background: white;
	font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
}
.chat-message {
	margin-bottom: 0.75rem;
	padding: 0.75rem;
	background: #f3f4f6;
	border-radius: 0.375rem;
}
.chat-username {
	font-weight: bold;
	color: #2563eb;
	margin-right: 0.5rem;
}
.chat-time {
	color: #6b7280;
	font-size: 0.875rem;
}
.chat-text {
	margin-top: 0.25rem;
}
.system-message {
	text-align: center;
	color: #6b7280;
	font-style: italic;
	margin: 1rem 0;
}
</style>`)
	stream.WriteString("</head>\n<body>\n<div id=\"messages\">\n")

	// Send welcome message first
	stream.WriteNode(h.Div(h.Class("system-message"), 
		g.Text("Connected to chat room. New messages will appear automatically."),
	))

	// Send existing messages
	messages := chatRoom.GetMessages()
	for _, msg := range messages {
		stream.WriteNode(renderChatMessageNode(msg))
	}

	// Keep connection alive and stream new messages
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case msg := <-msgChan:
			stream.WriteNode(renderChatMessageNode(msg))
			
		case <-ticker.C:
			// Send keep-alive
			stream.KeepAlive()
			
		case <-ctx.Request.Context().Done():
			// Client disconnected
			stream.WriteString("</div></body></html>")
			return nil
		}
	}
}

// ChatRoom methods

func (cr *ChatRoom) AddMessage(username, message string) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	
	msg := ChatMessage{
		ID:        cr.nextID,
		Username:  username,
		Message:   message,
		Timestamp: time.Now(),
	}
	cr.nextID++
	
	cr.messages = append(cr.messages, msg)
	
	// Broadcast to all subscribers
	for _, ch := range cr.subscribers {
		select {
		case ch <- msg:
		default:
			// Skip if channel is full
		}
	}
}

func (cr *ChatRoom) GetMessages() []ChatMessage {
	cr.mu.RLock()
	defer cr.mu.RUnlock()
	
	// Return last 50 messages
	start := 0
	if len(cr.messages) > 50 {
		start = len(cr.messages) - 50
	}
	
	result := make([]ChatMessage, len(cr.messages[start:]))
	copy(result, cr.messages[start:])
	return result
}

func (cr *ChatRoom) Subscribe(id string) chan ChatMessage {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	
	ch := make(chan ChatMessage, 10)
	cr.subscribers[id] = ch
	return ch
}

func (cr *ChatRoom) Unsubscribe(id string) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	
	if ch, exists := cr.subscribers[id]; exists {
		close(ch)
		delete(cr.subscribers, id)
	}
}

// Helper functions

func addTodo(text string) {
	todos[nextID] = &TodoItem{
		ID:        nextID,
		Text:      text,
		Completed: false,
		CreatedAt: time.Now(),
	}
	nextID++
}

func renderTodoList() g.Node {
	if len(todos) == 0 {
		return h.P(h.Class("empty"), g.Text("No todos yet. Add one to get started!"))
	}

	var items []g.Node
	for _, todo := range todos {
		class := "todo-item"
		if todo.Completed {
			class += " completed"
		}

		items = append(items, h.Div(h.Class(class),
			h.Div(h.Class("todo-content"),
				h.Span(g.Text(todo.Text)),
				h.Small(g.Text(nojs.TimeSince(todo.CreatedAt))),
			),
			h.Div(h.Class("todo-actions"),
				nojs.Form(nojs.FormConfig{
					Action: "/todos/toggle",
					Method: "POST",
					Class:  "inline-form",
				},
					h.Input(h.Type("hidden"), h.Name("id"), h.Value(fmt.Sprintf("%d", todo.ID))),
					h.Button(h.Type("submit"), h.Class("button-small"),
						g.If(todo.Completed, g.Text("Undo")),
						g.If(!todo.Completed, g.Text("Complete")),
					),
				),
				nojs.Form(nojs.FormConfig{
					Action: "/todos/delete",
					Method: "POST",
					Class:  "inline-form",
				},
					h.Input(h.Type("hidden"), h.Name("id"), h.Value(fmt.Sprintf("%d", todo.ID))),
					h.Button(h.Type("submit"), h.Class("button-small button-danger"),
						g.Text("Delete"),
					),
				),
			),
		))
	}

	todoListItems := append([]g.Node{h.Class("todo-list")}, items...)
	return h.Div(todoListItems...)
}

func renderAddModal() g.Node {
	return h.Div(h.Class("modal-backdrop"),
		h.Div(h.Class("modal"),
			h.Div(h.Class("modal-header"),
				h.H2(g.Text("Add New Todo")),
				h.A(h.Href("/todos"), h.Class("close"), g.Text("×")),
			),
			h.Div(h.Class("modal-body"),
				nojs.Form(nojs.FormConfig{
					Action: "/todos/add",
					Method: "POST",
				},
					h.Div(h.Class("form-group"),
						h.Label(h.For("text"), g.Text("Todo Text")),
						h.Input(
							h.Type("text"),
							h.Name("text"),
							h.ID("text"),
							h.Placeholder("Enter your todo..."),
							h.Required(),
							h.AutoFocus(),
						),
					),
					h.Div(h.Class("form-actions"),
						h.A(h.Href("/todos"), h.Class("button button-secondary"), g.Text("Cancel")),
						h.Button(h.Type("submit"), h.Class("button"), g.Text("Add Todo")),
					),
				),
			),
		),
	)
}

func renderUsernameForm() g.Node {
	return nojs.Card("Choose a Username", 
		nojs.Form(nojs.FormConfig{
			Action: "/chat",
			Method: "POST",
		},
			nojs.Input("Username", "username", "text", "", 
				h.Required(),
				h.AutoFocus(),
				h.Placeholder("Enter your username..."),
			),
			nojs.SubmitButton("Join Chat"),
		),
	)
}

func renderChatMessageNode(msg ChatMessage) g.Node {
	return h.Div(h.Class("chat-message"),
		h.Div(
			h.Span(h.Class("chat-username"), g.Text(msg.Username)),
			h.Span(h.Class("chat-time"), g.Text(msg.Timestamp.Format("15:04:05"))),
		),
		h.Div(h.Class("chat-text"), g.Text(msg.Message)),
	)
}