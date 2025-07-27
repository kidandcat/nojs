package nojs

import (
	"fmt"
	"strings"

	g "maragu.dev/gomponents"
	c "maragu.dev/gomponents/components"
	h "maragu.dev/gomponents/html"
)

// Page represents a full HTML page
type Page struct {
	Title       string
	Description string
	CSS         []string
	Body        g.Node
	Scripts     []g.Node // For progressive enhancement only
}

// Render renders a complete HTML page
func (p Page) Render(nodes ...g.Node) g.Node {
	return c.HTML5(
		c.HTML5Props{
			Title:       p.Title,
			Description: p.Description,
			Head: []g.Node{
				h.Meta(h.Charset("utf-8")),
				h.Meta(h.Name("viewport"), h.Content("width=device-width, initial-scale=1")),
				g.Map(p.CSS, func(css string) g.Node {
					return h.Link(h.Rel("stylesheet"), h.Href(css))
				}),
			},
			Body: append([]g.Node{p.Body}, append(nodes, p.Scripts...)...),
		},
	)
}

// Layout represents a reusable page layout
type Layout struct {
	Title       string
	CSS         []string
	Header      g.Node
	Navigation  g.Node
	Footer      g.Node
}

// Wrap wraps content in the layout
func (l Layout) Wrap(content g.Node) g.Node {
	return Page{
		Title: l.Title,
		CSS:   l.CSS,
		Body: h.Body(
			g.If(l.Header != nil, l.Header),
			g.If(l.Navigation != nil, l.Navigation),
			h.Main(content),
			g.If(l.Footer != nil, l.Footer),
		),
	}.Render()
}

// Form helpers for no-JS forms

// FormConfig configures a form
type FormConfig struct {
	Action   string
	Method   string
	Class    string
	Redirect string // For post-submit redirect
}

// Form creates a form with proper no-JS handling
func Form(config FormConfig, children ...g.Node) g.Node {
	method := config.Method
	if method == "" {
		method = "POST"
	}

	nodes := []g.Node{
		h.Action(config.Action),
		h.Method(method),
	}

	if config.Class != "" {
		nodes = append(nodes, h.Class(config.Class))
	}

	// Add method override for DELETE, PUT, etc.
	if method != "GET" && method != "POST" {
		children = append([]g.Node{
			h.Input(h.Type("hidden"), h.Name("_method"), h.Value(method)),
		}, children...)
		nodes[1] = h.Method("POST")
	}

	// Add redirect field if specified
	if config.Redirect != "" {
		children = append([]g.Node{
			h.Input(h.Type("hidden"), h.Name("_redirect"), h.Value(config.Redirect)),
		}, children...)
	}

	return h.Form(append(nodes, children...)...)
}

// Input creates an input with label
func Input(label, name, inputType, value string, attrs ...g.Node) g.Node {
	id := "input-" + name
	input := h.Input(
		append([]g.Node{
			h.Type(inputType),
			h.Name(name),
			h.ID(id),
			h.Value(value),
		}, attrs...)...,
	)

	if label == "" {
		return input
	}

	return h.Div(h.Class("form-group"),
		h.Label(h.For(id), g.Text(label)),
		input,
	)
}

// Select creates a select dropdown with label
func Select(label, name string, options []Option, selected string, attrs ...g.Node) g.Node {
	id := "select-" + name
	selectAttrs := append([]g.Node{h.Name(name), h.ID(id)}, attrs...)

	selectNode := h.Select(
		append(selectAttrs, func() []g.Node {
			var nodes []g.Node
			for _, opt := range options {
				optAttrs := []g.Node{h.Value(opt.Value)}
				if opt.Value == selected {
					optAttrs = append(optAttrs, h.Selected())
				}
				nodes = append(nodes, h.Option(append(optAttrs, g.Text(opt.Label))...))
			}
			return nodes
		}()...)...,
	)

	if label == "" {
		return selectNode
	}

	return h.Div(h.Class("form-group"),
		h.Label(h.For(id), g.Text(label)),
		selectNode,
	)
}

// Option represents a select option
type Option struct {
	Value string
	Label string
}

// Button creates a button
func Button(text string, attrs ...g.Node) g.Node {
	return h.Button(append(attrs, g.Text(text))...)
}

// SubmitButton creates a submit button
func SubmitButton(text string, attrs ...g.Node) g.Node {
	return Button(text, append([]g.Node{h.Type("submit")}, attrs...)...)
}

// Card creates a card component
func Card(title string, content g.Node, attrs ...g.Node) g.Node {
	nodes := append([]g.Node{h.Class("card")}, attrs...)
	
	if title != "" {
		nodes = append(nodes, h.Div(h.Class("card-header"), h.H3(g.Text(title))))
	}
	
	nodes = append(nodes, h.Div(h.Class("card-body"), content))
	
	return h.Div(nodes...)
}

// Alert creates an alert message
func Alert(message string, alertType string) g.Node {
	class := "alert"
	if alertType != "" {
		class += " alert-" + alertType
	}
	return h.Div(h.Class(class), g.Text(message))
}

// Table creates a responsive table
func Table(headers []string, rows [][]string, attrs ...g.Node) g.Node {
	tableAttrs := append([]g.Node{h.Class("table")}, attrs...)
	
	headerNodes := []g.Node{}
	for _, header := range headers {
		headerNodes = append(headerNodes, h.Th(g.Text(header)))
	}
	
	bodyNodes := []g.Node{}
	for _, row := range rows {
		rowNodes := []g.Node{}
		for _, cell := range row {
			rowNodes = append(rowNodes, h.Td(g.Text(cell)))
		}
		bodyNodes = append(bodyNodes, h.Tr(rowNodes...))
	}
	
	return h.Div(h.Class("table-responsive"),
		h.Table(
			append(tableAttrs,
				h.THead(h.Tr(headerNodes...)),
				h.TBody(bodyNodes...),
			)...,
		),
	)
}

// Modal creates a modal that works without JavaScript
func Modal(id string, title string, content g.Node, showParam string) g.Node {
	// The modal is shown when the URL contains ?modal=id or ?showParam=id
	return h.Div(
		h.ID(id),
		h.Class("modal"),
		h.Div(h.Class("modal-backdrop"),
			h.A(h.Href("#"), h.Class("modal-close")),
		),
		h.Div(h.Class("modal-content"),
			h.Div(h.Class("modal-header"),
				h.H2(g.Text(title)),
				h.A(h.Href("#"), h.Class("modal-close"), g.Text("×")),
			),
			h.Div(h.Class("modal-body"), content),
		),
	)
}

// AutoRefresh adds meta refresh tag for periodic updates
func AutoRefresh(seconds int) g.Node {
	return h.Meta(
		g.Attr("http-equiv", "refresh"),
		g.Attr("content", fmt.Sprintf("%d", seconds)),
	)
}

// Navigation creates a navigation menu
func Navigation(links []NavLink, currentPath string) g.Node {
	var items []g.Node
	for _, link := range links {
		classes := []string{"nav-item"}
		if link.Path == currentPath {
			classes = append(classes, "active")
		}
		items = append(items, h.Li(h.Class(strings.Join(classes, " ")),
			h.A(h.Href(link.Path), g.Text(link.Label)),
		))
	}
	
	navItems := append([]g.Node{h.Class("nav-list")}, items...)
	return h.Nav(h.Class("navbar"),
		h.Ul(navItems...),
	)
}

// NavLink represents a navigation link
type NavLink struct {
	Path  string
	Label string
}

// Pagination creates pagination controls
func Pagination(currentPage, totalPages int, baseURL string) g.Node {
	if totalPages <= 1 {
		return nil
	}

	var items []g.Node

	// Previous button
	if currentPage > 1 {
		items = append(items, h.Li(h.Class("page-item"),
			h.A(h.Class("page-link"), h.Href(fmt.Sprintf("%s?page=%d", baseURL, currentPage-1)), g.Text("Previous")),
		))
	}

	// Page numbers
	for i := 1; i <= totalPages; i++ {
		if i == currentPage {
			items = append(items, h.Li(h.Class("page-item active"),
				h.Span(h.Class("page-link"), g.Text(fmt.Sprintf("%d", i))),
			))
		} else {
			items = append(items, h.Li(h.Class("page-item"),
				h.A(h.Class("page-link"), h.Href(fmt.Sprintf("%s?page=%d", baseURL, i)), g.Text(fmt.Sprintf("%d", i))),
			))
		}
	}

	// Next button
	if currentPage < totalPages {
		items = append(items, h.Li(h.Class("page-item"),
			h.A(h.Class("page-link"), h.Href(fmt.Sprintf("%s?page=%d", baseURL, currentPage+1)), g.Text("Next")),
		))
	}

	paginationItems := append([]g.Node{h.Class("pagination")}, items...)
	return h.Nav(h.Ul(paginationItems...))
}