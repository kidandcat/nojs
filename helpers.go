package nojs

import (
	"fmt"
	"html/template"
	"strings"
	"time"
	
	g "maragu.dev/gomponents"
)

// FormatTime formats time in a human-readable way
func FormatTime(t time.Time) string {
	return t.Format("3:04 PM")
}

// FormatDate formats date in a human-readable way
func FormatDate(t time.Time) string {
	return t.Format("January 2, 2006")
}

// FormatDateTime formats date and time
func FormatDateTime(t time.Time) string {
	return t.Format("January 2, 2006 at 3:04 PM")
}

// TimeSince returns time elapsed since the given time
func TimeSince(t time.Time) string {
	duration := time.Since(t)
	
	if duration < time.Minute {
		return "just now"
	} else if duration < time.Hour {
		minutes := int(duration.Minutes())
		if minutes == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	} else if duration < 24*time.Hour {
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	} else if duration < 7*24*time.Hour {
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
	
	return FormatDate(t)
}

// Truncate truncates text to a maximum length
func Truncate(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}
	return text[:maxLength-3] + "..."
}

// Pluralize returns singular or plural form based on count
func Pluralize(count int, singular, plural string) string {
	if count == 1 {
		return fmt.Sprintf("%d %s", count, singular)
	}
	return fmt.Sprintf("%d %s", count, plural)
}

// FormatBytes formats bytes in human-readable format
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// SanitizeHTML escapes HTML to prevent XSS
func SanitizeHTML(text string) template.HTML {
	// Basic HTML escaping
	text = strings.ReplaceAll(text, "&", "&amp;")
	text = strings.ReplaceAll(text, "<", "&lt;")
	text = strings.ReplaceAll(text, ">", "&gt;")
	text = strings.ReplaceAll(text, "\"", "&quot;")
	text = strings.ReplaceAll(text, "'", "&#39;")
	return template.HTML(text)
}

// BuildURL builds a URL with query parameters
func BuildURL(base string, params map[string]string) string {
	if len(params) == 0 {
		return base
	}
	
	var parts []string
	for key, value := range params {
		parts = append(parts, fmt.Sprintf("%s=%s", key, value))
	}
	
	separator := "?"
	if strings.Contains(base, "?") {
		separator = "&"
	}
	
	return base + separator + strings.Join(parts, "&")
}

// Contains checks if a slice contains a value
func Contains[T comparable](slice []T, value T) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}

// Map applies a function to each element of a slice
func Map[T, U any](slice []T, fn func(T) U) []U {
	result := make([]U, len(slice))
	for i, v := range slice {
		result[i] = fn(v)
	}
	return result
}

// Filter returns elements that match a predicate
func Filter[T any](slice []T, predicate func(T) bool) []T {
	var result []T
	for _, v := range slice {
		if predicate(v) {
			result = append(result, v)
		}
	}
	return result
}

// GroupBy groups elements by a key function
func GroupBy[T any, K comparable](slice []T, keyFn func(T) K) map[K][]T {
	result := make(map[K][]T)
	for _, v := range slice {
		key := keyFn(v)
		result[key] = append(result[key], v)
	}
	return result
}

// Style creates a style element with CSS content
func Style(css string) g.Node {
	return g.Raw("<style>" + css + "</style>")
}

// Styles creates a style element from a map of CSS rules
func Styles(rules map[string]map[string]string) g.Node {
	var css strings.Builder
	css.WriteString("<style>\n")
	for selector, properties := range rules {
		css.WriteString(selector)
		css.WriteString(" {\n")
		for prop, value := range properties {
			css.WriteString(fmt.Sprintf("\t%s: %s;\n", prop, value))
		}
		css.WriteString("}\n")
	}
	css.WriteString("</style>")
	return g.Raw(css.String())
}