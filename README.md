# NoJS - The Modern No-JavaScript Web Framework

NoJS is a modern Go web framework built from the ground up to work completely without JavaScript. Every feature, every interaction, every component works perfectly with JavaScript disabled. This isn't a fallback mode or progressive enhancement - this is the core design principle.

## 100% JavaScript-Free by Design

NoJS doesn't just work without JavaScript - it's engineered specifically for JavaScript-free applications. Every single feature is built to function perfectly with JavaScript completely disabled in the browser.

- **Zero JavaScript Required**: Not optional, not progressive - completely JavaScript-free
- **Full Functionality**: Every interaction, animation, and update works without any client-side scripting
- **Modern Experience**: Proves that modern web apps don't need JavaScript
- **Always Works**: No fallbacks needed because there's nothing to fall back from

## Why NoJS?

In a world of bloated JavaScript frameworks, NoJS takes a radical approach: what if we didn't need JavaScript at all? The result is:

- **Instant Load Times**: No bundles to download, parse, or execute
- **Perfect Accessibility**: Works on every device, every browser, every assistive technology
- **Unbreakable**: No JavaScript means no JavaScript errors
- **Secure by Default**: No XSS attacks, no client-side vulnerabilities
- **SEO Perfect**: Search engines see exactly what users see

## Core Features

### Everything Works Without JavaScript
- **Real-time Updates**: HTML streaming for live data
- **Interactive Forms**: Full validation and dynamic behavior
- **Modal Dialogs**: CSS-powered overlays and popups
- **Data Tables**: Sorting, filtering, and pagination
- **Navigation**: Dropdowns, menus, and tabs
- **File Uploads**: With progress indication
- **Auto-refresh**: For dashboards and monitoring

### Modern Developer Experience
- **Type-Safe HTML**: Compile-time checks with Gomponents
- **Hot Reloading**: Fast development cycle
- **Component Library**: Pre-built UI components
- **Middleware System**: Extensible request pipeline
- **Session Management**: Built-in auth and flash messages

## Installation

```bash
go get github.com/jairo/nojs
```

## Quick Start

```go
package main

import (
    "log"
    "github.com/jairo/nojs"
    g "maragu.dev/gomponents"
    h "maragu.dev/gomponents/html"
)

func main() {
    server := nojs.NewServer()
    
    // Every feature works with JavaScript disabled
    server.Use(nojs.Logger())
    server.Use(nojs.Recovery())
    
    server.Route("/", handleHome)
    server.Route("/dashboard", handleDashboard)
    
    log.Fatal(server.Start(":8080"))
}

func handleHome(ctx *nojs.Context) error {
    // This page works 100% without JavaScript
    page := nojs.Page{
        Title: "NoJS - No JavaScript Required",
        Body: h.Div(
            h.H1(g.Text("Modern Web Apps Without JavaScript")),
            h.P(g.Text("Every feature on this site works with JavaScript disabled")),
            nojs.Button("Try Me", "/demo"),
        ),
    }
    return ctx.HTML(200, page.Render())
}
```

## Real-World Examples

### Live Updates Without JavaScript

```go
// Real-time streaming - no WebSockets, no JavaScript
func handleLiveStream(ctx *nojs.Context) error {
    stream, _ := ctx.Stream()
    
    stream.StartHTML("Live Feed")
    stream.WriteHTML("<div id='feed'>")
    
    // Updates appear instantly without any JavaScript
    for event := range eventChannel {
        stream.WriteHTML(renderEvent(event))
        stream.Flush()
    }
    
    stream.WriteHTML("</div>")
    return stream.EndHTML()
}
```

### Interactive Components Without JavaScript

```go
// Modal dialog - pure CSS, no JavaScript
nojs.Modal("user-form", "Create User",
    nojs.Form(nojs.FormConfig{
        Action: "/users",
        Method: "POST",
    },
        nojs.Input("Name", "name", "text", ""),
        nojs.Input("Email", "email", "email", ""),
        nojs.SubmitButton("Create"),
    ),
)

// Dropdown menu - works without JavaScript
nojs.Dropdown("Options",
    nojs.MenuItem("Profile", "/profile"),
    nojs.MenuItem("Settings", "/settings"),
    nojs.MenuItem("Logout", "/logout"),
)

// Tab navigation - no JavaScript needed
nojs.Tabs([]nojs.Tab{
    {ID: "overview", Label: "Overview", Content: overviewContent},
    {ID: "details", Label: "Details", Content: detailsContent},
    {ID: "history", Label: "History", Content: historyContent},
})
```

### Auto-Refreshing Dashboards

```go
// Dashboard updates every 5 seconds - no JavaScript
func handleDashboard(ctx *nojs.Context) error {
    return ctx.HTML(200, nojs.Page{
        Title: "Live Dashboard",
        Body: h.Div(
            nojs.AutoRefresh(5), // Meta refresh, not JavaScript
            h.H1(g.Text("System Status")),
            renderMetrics(getCurrentMetrics()),
        ),
    }.Render())
}
```

## How It Works

NoJS achieves full functionality without JavaScript through:

1. **HTML Forms**: All interactions use standard form submissions
2. **CSS Interactions**: Checkboxes and :target selectors for UI state
3. **Meta Refresh**: Auto-updating pages without scripts
4. **HTTP Streaming**: Real-time updates via chunked responses
5. **Server State**: All logic runs on the server

## Component Library

Complete UI toolkit that works without JavaScript:

```go
// Data grid with sorting and pagination
nojs.DataGrid(nojs.GridConfig{
    Data: users,
    Columns: []string{"Name", "Email", "Status"},
    Sortable: true,
    Paginated: true,
    PageSize: 20,
})

// File upload with progress
nojs.FileUpload(nojs.UploadConfig{
    Action: "/upload",
    Multiple: true,
    ShowProgress: true, // CSS-based progress
})

// Toast notifications
ctx.ShowToast("Success", "User created", "success")

// Accordion panels
nojs.Accordion([]nojs.Panel{
    {Title: "Section 1", Content: content1},
    {Title: "Section 2", Content: content2},
})
```

## Use Cases

NoJS is perfect when you need:

- **Maximum Compatibility**: Works on any browser, any device
- **Perfect Accessibility**: Screen readers and assistive tech just work
- **High Security**: No client-side attack surface
- **Fast Performance**: Instant page loads, no parsing
- **Simple Deployment**: Just HTML and CSS, no build process
- **SEO Excellence**: Full server-side rendering

## Not Just a Fallback

This is important: NoJS isn't about graceful degradation or progressive enhancement. It's not a framework that works without JavaScript as a fallback. It's a framework designed from day one to deliver full functionality without any JavaScript whatsoever.

## Getting Started

1. Install NoJS
2. Build your app using server-side logic
3. Deploy anywhere - no build step required
4. Your app works perfectly with JavaScript disabled

That's it. No bundlers, no transpilers, no polyfills.

## Examples and Resources

- **Demo App**: [demo.nojs.dev](https://demo.nojs.dev) (try it with JavaScript disabled!)
- **Documentation**: [docs.nojs.dev](https://docs.nojs.dev)
- **Examples**: [github.com/jairo/nojs-examples](https://github.com/jairo/nojs-examples)
- **Community**: [discord.gg/nojs](https://discord.gg/nojs)

## License

MIT License - see LICENSE file for details

---

**NoJS**: Modern web development without JavaScript. Not as a fallback. Not as an option. As the only way.