package main

import (
	"fmt"
	"log"
	"os"

	"github.com/kidandcat/nojs"
	"github.com/jairo/mavis/nojs/demo/chat"
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

func main() {
	server := nojs.NewServer()

	// Add middleware
	server.Use(nojs.Logger())
	server.Use(nojs.Recovery())

	// Serve static files
	server.Static("/static/", "./static")

	// Landing page routes
	server.Route("/", landingPageHandler)
	server.Route("/features", featuresHandler)
	server.Route("/docs", docsHandler)
	server.Route("/github", githubRedirectHandler)

	// Register chat demo
	chatDemo := chat.NewChatDemo()
	chatDemo.RegisterRoutes(server, "/demo/chat")

	// Get port from environment or default to 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("🚀 NoJS Landing Page running on http://localhost:%s\n", port)
	log.Fatal(server.Start(":" + port))
}

func landingPageHandler(ctx *nojs.Context) error {
	page := nojs.Page{
		Title: "NoJS - The Modern No-JavaScript Web Framework",
		CSS:   []string{"/static/landing.css"},
		Body: h.Div(
			// Hero Section
			h.Section(h.Class("hero"),
				h.Div(h.Class("container"),
					h.H1(h.Class("hero-title"), g.Text("NoJS")),
					h.P(h.Class("hero-tagline"), g.Text("The Modern No-JavaScript Web Framework")),
					h.P(h.Class("hero-description"), 
						g.Text("Build blazing-fast web applications that work completely without JavaScript. "),
						g.Text("Every feature, every interaction, every component works perfectly with JavaScript disabled."),
					),
					h.Div(h.Class("hero-actions"),
						h.A(h.Href("/demo/chat"), h.Class("btn btn-primary"), g.Text("Try Live Demo")),
						h.A(h.Href("https://github.com/kidandcat/nojs"), h.Class("btn btn-secondary"), g.Text("View on GitHub")),
					),
				),
			),

			// Features Section
			h.Section(h.Class("features"),
				h.Div(h.Class("container"),
					h.H2(g.Text("100% JavaScript-Free by Design")),
					h.Div(h.Class("features-grid"),
						featureCard("⚡", "Instant Load Times", "No bundles to download, parse, or execute. Your app loads instantly."),
						featureCard("♿", "Perfect Accessibility", "Works on every device, every browser, every assistive technology."),
						featureCard("🔒", "Secure by Default", "No XSS attacks, no client-side vulnerabilities. Security built-in."),
						featureCard("🔍", "SEO Perfect", "Search engines see exactly what users see. Full server-side rendering."),
						featureCard("🚀", "Modern Experience", "Proves that modern web apps don't need JavaScript."),
						featureCard("🛡️", "Unbreakable", "No JavaScript means no JavaScript errors. Always works."),
					),
				),
			),

			// Demo Section
			h.Section(h.Class("demo-section"),
				h.Div(h.Class("container"),
					h.H2(g.Text("See It In Action")),
					h.P(h.Class("demo-description"), 
						g.Text("Our live chat demo showcases real-time updates, interactive forms, and modern UI - all without a single line of JavaScript."),
					),
					h.Div(h.Class("demo-preview"),
						h.Div(h.Class("demo-features"),
							h.H3(g.Text("Chat Demo Features:")),
							h.Ul(
								h.Li(g.Text("Real-time message streaming")),
								h.Li(g.Text("User identification with hashes")),
								h.Li(g.Text("Persistent sessions")),
								h.Li(g.Text("Colorful usernames")),
								h.Li(g.Text("Responsive design")),
								h.Li(g.Text("Zero JavaScript required")),
							),
						),
						h.Div(h.Class("demo-cta"),
							h.A(h.Href("/demo/chat"), h.Class("btn btn-large"), g.Text("Launch Chat Demo →")),
							h.P(h.Class("demo-note"), g.Text("Try it with JavaScript disabled in your browser!")),
						),
					),
				),
			),

			// Code Example Section
			h.Section(h.Class("code-section"),
				h.Div(h.Class("container"),
					h.H2(g.Text("Simple & Powerful")),
					h.Div(h.Class("code-example"),
						h.Pre(h.Code(g.Text(`package main

import (
    "log"
    "github.com/kidandcat/nojs"
    g "maragu.dev/gomponents"
    h "maragu.dev/gomponents/html"
)

func main() {
    server := nojs.NewServer()
    
    server.Route("/", func(ctx *nojs.Context) error {
        page := nojs.Page{
            Title: "Hello NoJS",
            Body: h.Div(
                h.H1(g.Text("Modern Web Without JavaScript")),
                h.P(g.Text("Every feature works with JS disabled")),
            ),
        }
        return ctx.HTML(200, page.Render())
    })
    
    log.Fatal(server.Start(":8080"))
}`))),
					),
				),
			),

			// CTA Section
			h.Section(h.Class("cta-section"),
				h.Div(h.Class("container"),
					h.H2(g.Text("Ready to Build Without JavaScript?")),
					h.P(g.Text("Join the movement towards simpler, faster, more accessible web applications.")),
					h.Div(h.Class("cta-actions"),
						h.A(h.Href("https://github.com/kidandcat/nojs"), h.Class("btn btn-primary"), g.Text("Get Started")),
						h.A(h.Href("/docs"), h.Class("btn btn-secondary"), g.Text("Read Documentation")),
					),
				),
			),

			// Footer
			h.Footer(h.Class("footer"),
				h.Div(h.Class("container"),
					h.P(g.Text("NoJS - Modern web development without JavaScript. Not as a fallback. Not as an option. As the only way.")),
					h.P(h.Class("footer-links"),
						h.A(h.Href("https://github.com/kidandcat/nojs"), g.Text("GitHub")),
						g.Text(" · "),
						h.A(h.Href("/docs"), g.Text("Documentation")),
						g.Text(" · "),
						h.A(h.Href("/demo/chat"), g.Text("Demo")),
					),
				),
			),
		),
	}

	return ctx.HTML(200, page.Render())
}

func featureCard(icon, title, description string) g.Node {
	return h.Div(h.Class("feature-card"),
		h.Div(h.Class("feature-icon"), g.Text(icon)),
		h.H3(g.Text(title)),
		h.P(g.Text(description)),
	)
}

func featuresHandler(ctx *nojs.Context) error {
	// Redirect to main page features section
	return ctx.Redirect(302, "/#features")
}

func docsHandler(ctx *nojs.Context) error {
	// Redirect to GitHub README for now
	return ctx.Redirect(302, "https://github.com/kidandcat/nojs#readme")
}

func githubRedirectHandler(ctx *nojs.Context) error {
	return ctx.Redirect(302, "https://github.com/kidandcat/nojs")
}