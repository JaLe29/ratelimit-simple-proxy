package middleware

import (
	"bytes"
	"net/http"
	"strings"
)

// HTMLInjectMiddleware injects a control panel into HTML responses
type HTMLInjectMiddleware struct {
	next             http.Handler
	controlPanelHTML string
}

// NewHTMLInjectMiddleware creates a new HTML injection middleware
func NewHTMLInjectMiddleware(next http.Handler, controlPanelHTML string) *HTMLInjectMiddleware {
	return &HTMLInjectMiddleware{
		next:             next,
		controlPanelHTML: controlPanelHTML,
	}
}

func (m *HTMLInjectMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Create a response writer that captures the response
	responseWriter := &responseCaptureWriter{
		ResponseWriter: w,
		buffer:         &bytes.Buffer{},
	}

	// Call the next handler
	m.next.ServeHTTP(responseWriter, r)

	// Check if the response is HTML
	if isHTMLResponse(responseWriter.Header()) {
		// Get the response body
		body := responseWriter.buffer.String()

		// Inject the control panel
		injectedBody := injectControlPanel(body, m.controlPanelHTML)

		// Write the modified response
		w.Header().Set("Content-Length", string(len(injectedBody)))
		w.Write([]byte(injectedBody))
	} else {
		// For non-HTML responses, write the original response
		w.Write(responseWriter.buffer.Bytes())
	}
}

// responseCaptureWriter captures the response body
type responseCaptureWriter struct {
	http.ResponseWriter
	buffer *bytes.Buffer
}

func (w *responseCaptureWriter) Write(data []byte) (int, error) {
	return w.buffer.Write(data)
}

// isHTMLResponse checks if the response is HTML
func isHTMLResponse(headers http.Header) bool {
	contentType := headers.Get("Content-Type")
	return strings.Contains(contentType, "text/html")
}

// injectControlPanel injects the control panel into the HTML body
func injectControlPanel(html, controlPanelHTML string) string {
	// Find the closing </body> tag and inject the control panel before it
	if strings.Contains(html, "</body>") {
		return strings.Replace(html, "</body>", controlPanelHTML+"</body>", 1)
	}

	// If no </body> tag found, append to the end
	return html + controlPanelHTML
}
