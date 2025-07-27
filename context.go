package nojs

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// Context represents the request context
type Context struct {
	Request        *http.Request
	ResponseWriter http.ResponseWriter
	server         *Server
	params         map[string]string
	written        bool
}

// Handler is a function that handles HTTP requests
type Handler func(*Context) error

// Middleware is a function that wraps a handler
type Middleware func(Handler) Handler

// Param returns a URL parameter by name
func (c *Context) Param(name string) string {
	if c.params == nil {
		return ""
	}
	return c.params[name]
}

// Query returns a query parameter by name
func (c *Context) Query(name string) string {
	return c.Request.URL.Query().Get(name)
}

// QueryValues returns all values for a query parameter
func (c *Context) QueryValues(name string) []string {
	return c.Request.URL.Query()[name]
}

// Form returns a form value by name
func (c *Context) Form(name string) string {
	if c.Request.Method == "POST" || c.Request.Method == "PUT" {
		c.Request.ParseForm()
	}
	return c.Request.FormValue(name)
}

// FormValues returns all values for a form field
func (c *Context) FormValues(name string) []string {
	if c.Request.Method == "POST" || c.Request.Method == "PUT" {
		c.Request.ParseForm()
	}
	return c.Request.Form[name]
}

// JSON sends a JSON response
func (c *Context) JSON(status int, data interface{}) error {
	c.ResponseWriter.Header().Set("Content-Type", "application/json")
	c.ResponseWriter.WriteHeader(status)
	c.written = true
	return json.NewEncoder(c.ResponseWriter).Encode(data)
}

// HTML renders an HTML response using gomponents
func (c *Context) HTML(status int, node g.Node) error {
	c.ResponseWriter.Header().Set("Content-Type", "text/html; charset=utf-8")
	c.ResponseWriter.WriteHeader(status)
	c.written = true
	return node.Render(c.ResponseWriter)
}

// Text sends a plain text response
func (c *Context) Text(status int, text string) error {
	c.ResponseWriter.Header().Set("Content-Type", "text/plain; charset=utf-8")
	c.ResponseWriter.WriteHeader(status)
	c.written = true
	_, err := c.ResponseWriter.Write([]byte(text))
	return err
}

// Redirect redirects the request
func (c *Context) Redirect(status int, url string) error {
	http.Redirect(c.ResponseWriter, c.Request, url, status)
	c.written = true
	return nil
}

// Stream enables HTTP streaming for real-time updates
func (c *Context) Stream() (*StreamWriter, error) {
	if !c.server.config.StreamingEnabled {
		return nil, NewHTTPError(http.StatusInternalServerError, "Streaming not enabled")
	}

	flusher, ok := c.ResponseWriter.(http.Flusher)
	if !ok {
		return nil, NewHTTPError(http.StatusInternalServerError, "Streaming not supported")
	}

	// Set headers for streaming
	c.ResponseWriter.Header().Set("Content-Type", "text/html; charset=utf-8")
	c.ResponseWriter.Header().Set("Cache-Control", "no-cache")
	c.ResponseWriter.Header().Set("X-Content-Type-Options", "nosniff")

	return &StreamWriter{
		writer:  c.ResponseWriter,
		flusher: flusher,
		context: c,
	}, nil
}

// SetFlash sets a flash message (stored in cookie)
func (c *Context) SetFlash(name, value string) {
	http.SetCookie(c.ResponseWriter, &http.Cookie{
		Name:     "flash_" + name,
		Value:    url.QueryEscape(value),
		Path:     "/",
		MaxAge:   60, // 60 seconds
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
}

// GetFlash gets and removes a flash message
func (c *Context) GetFlash(name string) string {
	cookie, err := c.Request.Cookie("flash_" + name)
	if err != nil {
		return ""
	}

	// Clear the flash message
	http.SetCookie(c.ResponseWriter, &http.Cookie{
		Name:     "flash_" + name,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})

	value, _ := url.QueryUnescape(cookie.Value)
	return value
}

// IsHTMX returns true if the request is from HTMX (for progressive enhancement)
func (c *Context) IsHTMX() bool {
	return c.Request.Header.Get("HX-Request") == "true"
}

// IsJSON returns true if the request expects JSON response
func (c *Context) IsJSON() bool {
	accept := c.Request.Header.Get("Accept")
	contentType := c.Request.Header.Get("Content-Type")
	return strings.Contains(accept, "application/json") || strings.Contains(contentType, "application/json")
}

// Method returns the HTTP method
func (c *Context) Method() string {
	// Check for method override
	if c.Request.Method == "POST" {
		if override := c.Form("_method"); override != "" {
			return strings.ToUpper(override)
		}
	}
	return c.Request.Method
}

// StreamWriter handles HTTP streaming responses
type StreamWriter struct {
	writer  http.ResponseWriter
	flusher http.Flusher
	context *Context
}

// Write writes data to the stream and flushes immediately
func (sw *StreamWriter) Write(data []byte) (int, error) {
	n, err := sw.writer.Write(data)
	sw.flusher.Flush()
	return n, err
}

// WriteString writes a string to the stream
func (sw *StreamWriter) WriteString(s string) error {
	_, err := sw.Write([]byte(s))
	return err
}

// WriteHTML writes HTML content to the stream
func (sw *StreamWriter) WriteHTML(html string) error {
	return sw.WriteString(html)
}

// WriteNode writes a gomponents node to the stream
func (sw *StreamWriter) WriteNode(nodes ...g.Node) error {
	for _, node := range nodes {
		if err := node.Render(sw); err != nil {
			return err
		}
	}
	return nil
}

// KeepAlive sends a keep-alive comment to prevent timeout
func (sw *StreamWriter) KeepAlive() error {
	return sw.WriteString("<!-- keepalive -->\n")
}

// StartHTML writes the beginning of an HTML document for streaming using gomponents
func (sw *StreamWriter) StartHTML(title string, headNodes ...g.Node) error {
	// We need to write the opening tags but not close body/html
	html := `<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<title>` + title + `</title>
<meta name="viewport" content="width=device-width, initial-scale=1">`
	
	for _, node := range headNodes {
		var buf strings.Builder
		node.Render(&buf)
		html += buf.String()
	}
	
	html += `
</head>
<body>
`
	return sw.WriteString(html)
}

// EndHTML writes the end of an HTML document
func (sw *StreamWriter) EndHTML() error {
	return sw.WriteString("</body>\n</html>\n")
}

// StreamPage starts streaming an HTML page with the given configuration
func (sw *StreamWriter) StreamPage(title string, css []string) error {
	headNodes := []g.Node{}
	for _, cssPath := range css {
		headNodes = append(headNodes, h.Link(h.Rel("stylesheet"), h.Href(cssPath)))
	}
	return sw.StartHTML(title, headNodes...)
}

// Sleep pauses execution while maintaining the connection
func (sw *StreamWriter) Sleep(duration time.Duration) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	timer := time.NewTimer(duration)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			return
		case <-ticker.C:
			sw.KeepAlive()
		case <-sw.context.Request.Context().Done():
			return
		}
	}
}