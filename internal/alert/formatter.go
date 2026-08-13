package alert

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Formatter renders an Alert into a webhook payload plus any extra headers
// required by the target platform.
type Formatter interface {
	// ContentType is the HTTP Content-Type of the payload.
	ContentType() string
	// Headers are extra HTTP headers (e.g. Gotify's X-Gotify-Key).
	Headers() map[string]string
	// Format renders the alert body.
	Format(a Alert) ([]byte, error)
}

// ---------------------------------------------------------------------------
// Feishu (Lark) custom bot webhook
// ---------------------------------------------------------------------------

// LarkFormatter renders alerts for a Feishu custom bot webhook
// (https://open.feishu.cn/document/client-docs/bot-v3/add-custom-bot).
type LarkFormatter struct{}

// NewLarkFormatter creates a Feishu formatter.
func NewLarkFormatter() *LarkFormatter { return &LarkFormatter{} }

func (f *LarkFormatter) ContentType() string { return "application/json" }
func (f *LarkFormatter) Headers() map[string]string {
	return map[string]string{}
}

// Format produces the Feishu "text" message payload:
//
//	{"msg_type":"text","content":{"text":"..."}}
func (f *LarkFormatter) Format(a Alert) ([]byte, error) {
	text := fmt.Sprintf("【%s】%s\n%s\n服务: %s\n时间: %s",
		strings.ToUpper(a.Level.String()), a.Title, a.Message, a.Service,
		a.Time.Format("2006-01-02 15:04:05"),
	)
	if len(a.Fields) > 0 {
		text += "\n---"
		for k, v := range a.Fields {
			text += fmt.Sprintf("\n%s: %s", k, v)
		}
	}

	return json.Marshal(map[string]any{
		"msg_type": "text",
		"content":  map[string]string{"text": text},
	})
}

// ---------------------------------------------------------------------------
// Gotify push messages
// ---------------------------------------------------------------------------

// GotifyFormatter renders alerts for a Gotify server
// (https://gotify.net/docs/pushmsg), sent to POST /message with an app token
// in the X-Gotify-Key header. Gotify priority is 0..10.
type GotifyFormatter struct {
	token string
}

// NewGotifyFormatter creates a Gotify formatter for the given app token.
func NewGotifyFormatter(token string) *GotifyFormatter {
	return &GotifyFormatter{token: token}
}

func (f *GotifyFormatter) ContentType() string { return "application/json" }
func (f *GotifyFormatter) Headers() map[string]string {
	return map[string]string{"X-Gotify-Key": f.token}
}

// Format produces the Gotify message payload:
//
//	{"title":"...","message":"...","priority":8}
func (f *GotifyFormatter) Format(a Alert) ([]byte, error) {
	msg := a.Message
	if len(a.Fields) > 0 {
		var sb strings.Builder
		sb.WriteString(a.Message)
		for k, v := range a.Fields {
			sb.WriteString(fmt.Sprintf("\n%s: %s", k, v))
		}
		msg = sb.String()
	}

	return json.Marshal(map[string]any{
		"title":    fmt.Sprintf("[%s] %s", strings.ToUpper(a.Level.String()), a.Title),
		"message":  msg,
		"priority": gotifyPriority(a.Level),
	})
}

// gotifyPriority maps alert levels to Gotify priorities (0-10).
func gotifyPriority(l Level) int {
	switch l {
	case LevelInfo:
		return 0
	case LevelWarning:
		return 5
	case LevelCritical:
		return 10
	default:
		return 0
	}
}
