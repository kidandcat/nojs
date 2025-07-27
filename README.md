# NoJS - A Go Web Framework Without JavaScript

NoJS is a Go web framework for building modern web applications without client-side JavaScript. It embraces server-side rendering, HTML streaming, and web fundamentals to create fast, accessible, and maintainable web applications.

## Features

- **Zero JavaScript Required**: Build interactive web apps using only server-side Go
- **HTML Streaming**: Real-time updates using HTTP chunked transfer encoding
- **Gomponents Integration**: Type-safe HTML generation with compile-time checks
- **Form-First Design**: All interactions work with standard HTML forms
- **Flash Messages**: Server-side notifications without JavaScript
- **Auto-Refresh**: Periodic updates using meta refresh
- **Progressive Enhancement**: Can add JavaScript later if needed
- **Middleware System**: Extensible request processing pipeline
- **Built-in Components**: Forms, tables, cards, modals, and more

## Installation

```bash
go get github.com/jairo/mavis/nojs
```

## Quick Start

```go
package main

import (
    "log"
    "github.com/jairo/mavis/nojs"
    g "maragu.dev/gomponents"
    h "maragu.dev/gomponents/html"
)

func main() {
    // Create a new server
    server := nojs.NewServer()

    // Add middleware
    server.Use(nojs.Logger())
    server.Use(nojs.Recovery())

    // Define routes
    server.Route("/", handleHome)
    server.Route("/about", handleAbout)
    server.Route("/contact", handleContact)

    // Serve static files
    server.Static("/static/", "./static")

    // Start the server
    log.Fatal(server.Start(":8080"))
}

func handleHome(ctx *nojs.Context) error {
    page := nojs.Page{
        Title: "Home",
        Body: h.Div(
            h.H1(g.Text("Welcome to NoJS")),
            h.P(g.Text("Building web apps without JavaScript")),
        ),
    }
    return ctx.HTML(200, page.Render())
}

func handleAbout(ctx *nojs.Context) error {
    // Example with layout
    layout := nojs.Layout{
        Title: "About Us",
        Header: h.Header(
            h.Nav(h.A(h.Href("/"), g.Text("Home"))),
        ),
    }
    
    content := h.Div(
        h.H1(g.Text("About NoJS")),
        h.P(g.Text("NoJS is a framework for building web applications without client-side JavaScript.")),
    )
    
    return ctx.HTML(200, layout.Wrap(content))
}

func handleContact(ctx *nojs.Context) error {
    // Handle form submission
    if ctx.Method() == "POST" {
        name := ctx.Form("name")
        email := ctx.Form("email")
        message := ctx.Form("message")
        
        // Process the form data...
        
        ctx.SetFlash("success", "Thank you for your message!")
        return ctx.Redirect(303, "/contact")
    }
    
    // Show the form
    flash := ctx.GetFlash("success")
    
    content := h.Div(
        g.If(flash != "", nojs.Alert(flash, "success")),
        h.H1(g.Text("Contact Us")),
        nojs.Form(nojs.FormConfig{
            Action: "/contact",
            Method: "POST",
        },
            nojs.Input("Name", "name", "text", "", h.Required()),
            nojs.Input("Email", "email", "email", "", h.Required()),
            h.Div(h.Class("form-group"),
                h.Label(h.For("message"), g.Text("Message")),
                h.Textarea(h.Name("message"), h.ID("message"), h.Required()),
            ),
            nojs.SubmitButton("Send Message"),
        ),
    )
    
    return ctx.HTML(200, nojs.Page{Title: "Contact", Body: content}.Render())
}
```

## HTML Streaming Example

Real-time updates without WebSockets or Server-Sent Events:

```go
func handleStream(ctx *nojs.Context) error {
    stream, err := ctx.Stream()
    if err != nil {
        return err
    }
    
    // Start HTML document
    stream.StartHTML("Live Updates")
    
    // Send initial content
    stream.WriteHTML("<h1>Live Dashboard</h1>")
    stream.WriteHTML("<div id='updates'>")
    
    // Stream updates
    for i := 0; i < 10; i++ {
        update := fmt.Sprintf("<div>Update %d at %s</div>", i, time.Now().Format("15:04:05"))
        stream.WriteHTML(update)
        
        // Keep connection alive during processing
        stream.Sleep(2 * time.Second)
    }
    
    stream.WriteHTML("</div>")
    stream.EndHTML()
    
    return nil
}
```

## Components

NoJS includes pre-built components that work without JavaScript:

### Forms
```go
nojs.Form(nojs.FormConfig{
    Action: "/submit",
    Method: "POST",
},
    nojs.Input("Username", "username", "text", "", h.Required()),
    nojs.Input("Password", "password", "password", "", h.Required()),
    nojs.SubmitButton("Login"),
)
```

### Tables
```go
nojs.Table(
    []string{"Name", "Email", "Status"},
    [][]string{
        {"John Doe", "john@example.com", "Active"},
        {"Jane Smith", "jane@example.com", "Inactive"},
    },
)
```

### Cards
```go
nojs.Card("User Profile", 
    h.Div(
        h.P(g.Text("Name: John Doe")),
        h.P(g.Text("Email: john@example.com")),
    ),
)
```

### Modals (CSS-based, no JS)
```go
// Show modal with ?modal=create-user
nojs.Modal("create-user", "Create User", 
    nojs.Form(nojs.FormConfig{Action: "/users"},
        nojs.Input("Name", "name", "text", ""),
        nojs.SubmitButton("Create"),
    ),
    "modal",
)
```

## Middleware

Built-in middleware:

```go
// Logging
server.Use(nojs.Logger())

// Panic recovery
server.Use(nojs.Recovery())

// CORS
server.Use(nojs.CORS([]string{"*"}))

// Rate limiting
server.Use(nojs.RateLimit(100, time.Minute))

// Basic auth
server.Use(nojs.BasicAuth("Admin Area", map[string]string{
    "admin": "password",
}))

// No cache
server.Use(nojs.NoCache())
```

## Auto-Refresh Pages

For dashboards and real-time data:

```go
func handleDashboard(ctx *nojs.Context) error {
    page := nojs.Page{
        Title: "Dashboard",
        Body: h.Div(
            nojs.AutoRefresh(5), // Refresh every 5 seconds
            h.H1(g.Text("Live Dashboard")),
            h.P(g.Text(fmt.Sprintf("Last updated: %s", time.Now().Format("15:04:05")))),
            // ... dashboard content
        ),
    }
    return ctx.HTML(200, page.Render())
}
```

## Flash Messages

Server-side notifications:

```go
// Set a flash message
ctx.SetFlash("success", "Operation completed successfully!")
ctx.SetFlash("error", "Something went wrong")

// Display flash messages
flash := ctx.GetFlash("success")
if flash != "" {
    // Render alert
    nojs.Alert(flash, "success")
}
```

## Philosophy

NoJS embraces these principles:

1. **Simplicity**: Use web fundamentals (HTML, HTTP, Forms)
2. **Performance**: No JavaScript parsing or execution on client
3. **Accessibility**: Works everywhere, including screen readers
4. **SEO-Friendly**: Full server-side rendering
5. **Progressive Enhancement**: Add JavaScript only when truly needed
6. **Type Safety**: Compile-time HTML validation with gomponents

## When to Use NoJS

NoJS is perfect for:

- Admin dashboards
- CRUD applications  
- Internal tools
- Forms and surveys
- Content-heavy sites
- Applications where accessibility is critical
- Projects where simplicity is valued over complexity

## When NOT to Use NoJS

NoJS might not be suitable for:

- Complex interactive UIs (e.g., image editors, games)
- Applications requiring offline functionality
- Real-time collaboration tools with complex state
- Applications with heavy client-side computation

## License

MIT License - see LICENSE file for details