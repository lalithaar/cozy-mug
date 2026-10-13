// Package flash is a small one-shot message helper for cozy-mug apps (gap 7).
//
// It stores messages in the request session and exposes them through a processor.
// Messages are consumed on the first render after they are queued, so they work
// across redirects.
package flash

import (
	"encoding/json"

	"github.com/lalithaar/cozy-mug/core"
	"github.com/lalithaar/cozy-mug/session"
)

const defaultKey = "cozy_flash"

// Message is one queued flash message.
type Message struct {
	Level string `json:"level"`
	Text  string `json:"text"`
}

// Options tunes the session key used for storage.
type Options struct {
	Key string
}

func key(opts Options) string {
	if opts.Key != "" {
		return opts.Key
	}
	return defaultKey
}

// Add queues a message using the default session key.
func Add(c *core.Context, level, text string) {
	AddWithOptions(c, level, text, Options{})
}

// AddWithOptions queues a message with a custom storage key.
func AddWithOptions(c *core.Context, level, text string, opts Options) {
	sess := session.From(c)
	if sess == nil {
		return
	}
	messages := queued(sess, key(opts))
	messages = append(messages, Message{Level: level, Text: text})
	encoded, err := json.Marshal(messages)
	if err != nil {
		return
	}
	sess.Set(key(opts), string(encoded))
}

func queued(sess *session.Session, storageKey string) []Message {
	if sess == nil {
		return nil
	}
	value := sess.Get(storageKey)
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case string:
		var out []Message
		if err := json.Unmarshal([]byte(v), &out); err != nil {
			return nil
		}
		return out
	case []byte:
		var out []Message
		if err := json.Unmarshal(v, &out); err != nil {
			return nil
		}
		return out
	case []Message:
		out := make([]Message, len(v))
		copy(out, v)
		return out
	case []any:
		out := make([]Message, 0, len(v))
		for _, item := range v {
			msg, ok := toMessage(item)
			if ok {
				out = append(out, msg)
			}
		}
		return out
	default:
		return nil
	}
}

func toMessage(value any) (Message, bool) {
	switch v := value.(type) {
	case map[string]any:
		var msg Message
		if level, ok := v["Level"].(string); ok {
			msg.Level = level
		} else if level, ok := v["level"].(string); ok {
			msg.Level = level
		}
		if text, ok := v["Text"].(string); ok {
			msg.Text = text
		} else if text, ok := v["text"].(string); ok {
			msg.Text = text
		}
		return msg, true
	case Message:
		return v, true
	default:
		return Message{}, false
	}
}

// Messages consumes and returns the queued messages for the request.
func Messages(c *core.Context) []Message {
	return MessagesWithOptions(c, Options{})
}

// MessagesWithOptions consumes queued messages using a custom storage key.
func MessagesWithOptions(c *core.Context, opts Options) []Message {
	sess := session.From(c)
	if sess == nil {
		return nil
	}
	storageKey := key(opts)
	messages := queued(sess, storageKey)
	sess.Delete(storageKey)
	return messages
}

// Error, Warning, Info, and Success are level helpers.
func Error(c *core.Context, text string)   { Add(c, "error", text) }
func Warning(c *core.Context, text string) { Add(c, "warning", text) }
func Info(c *core.Context, text string)    { Add(c, "info", text) }
func Success(c *core.Context, text string) { Add(c, "success", text) }

// ErrorWithOptions and peers allow a custom storage key.
func ErrorWithOptions(c *core.Context, text string, opts Options) {
	AddWithOptions(c, "error", text, opts)
}
func WarningWithOptions(c *core.Context, text string, opts Options) {
	AddWithOptions(c, "warning", text, opts)
}
func InfoWithOptions(c *core.Context, text string, opts Options) {
	AddWithOptions(c, "info", text, opts)
}
func SuccessWithOptions(c *core.Context, text string, opts Options) {
	AddWithOptions(c, "success", text, opts)
}

// Processor returns a core.Processor exposing queued messages as
// {{.Values.flash}}.
func Processor() core.Processor {
	return ProcessorWithName("flash", Options{})
}

// ProcessorWithName exposes queued messages under a custom view value name.
func ProcessorWithName(name string, opts Options) core.Processor {
	if name == "" {
		name = "flash"
	}
	return func(c *core.Context) map[string]any {
		return map[string]any{name: MessagesWithOptions(c, opts)}
	}
}

// From returns the messages already popped by a processor, if stored by the app.
func From(c *core.Context) []Message {
	if c == nil {
		return nil
	}
	if value, ok := c.Global("flash").([]Message); ok {
		return value
	}
	if value := c.Global("flash"); value != nil {
		if v, ok := toMessageCollection(value); ok {
			return v
		}
	}
	return nil
}

func toMessageCollection(value any) ([]Message, bool) {
	switch v := value.(type) {
	case []Message:
		return v, true
	case []any:
		out := make([]Message, 0, len(v))
		for _, item := range v {
			msg, ok := toMessage(item)
			if !ok {
				return nil, false
			}
			out = append(out, msg)
		}
		return out, true
	default:
		return nil, false
	}
}
