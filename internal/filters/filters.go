package filters

import (
	"regexp"
	"strings"
	"sync"
)

// Config holds user-defined spam filter settings.
type Config struct {
	Blocklist             []string `json:"blocklist"`               // words/phrases to flag (case-insensitive)
	RepetitionThreshold   int      `json:"repetitionThreshold"`     // min repeats to count as spam (0 = disabled)
	LinkDetection         bool     `json:"linkDetection"`            // flag messages containing URLs
	CustomRegex           string   `json:"customRegex,omitempty"`   // optional regex to flag messages
	HideFiltered          bool     `json:"hideFiltered"`             // if true, hide spam; else show but mark
}

// Result is the outcome of running filters on a message.
type Result struct {
	IsSpam   bool     `json:"isSpam"`
	Reasons  []string `json:"reasons,omitempty"`
	Message  string   `json:"message"`  // original text
	AuthorID string   `json:"authorId"`
	MessageID string  `json:"messageId"`
}

var urlRegex = regexp.MustCompile(`(?i)(https?://[^\s]+)|(www\.[^\s]+)|([a-zA-Z0-9][-a-zA-Z0-9]*\.(com|org|net|io|co|tv|gg)[^\s]*)`)

// Engine applies spam filters to messages and tracks repetition.
type Engine struct {
	mu         sync.Mutex
	recentByAuthor map[string][]string // author ID -> recent message texts (for repetition)
	maxRecent  int
}

func NewEngine(maxRecentPerAuthor int) *Engine {
	if maxRecentPerAuthor <= 0 {
		maxRecentPerAuthor = 50
	}
	return &Engine{
		recentByAuthor: make(map[string][]string),
		maxRecent:      maxRecentPerAuthor,
	}
}

// Apply runs all enabled filters on the message text and returns the result.
func (e *Engine) Apply(cfg *Config, messageID, authorID, text string) Result {
	if cfg == nil {
		return Result{Message: text, AuthorID: authorID, MessageID: messageID}
	}
	var reasons []string
	lower := strings.ToLower(strings.TrimSpace(text))

	// Blocklist
	for _, word := range cfg.Blocklist {
		w := strings.ToLower(strings.TrimSpace(word))
		if w == "" {
			continue
		}
		if strings.Contains(lower, w) {
			reasons = append(reasons, "blocklist:"+word)
		}
	}

	// Link detection
	if cfg.LinkDetection && urlRegex.MatchString(text) {
		reasons = append(reasons, "link")
	}

	// Custom regex
	if cfg.CustomRegex != "" {
		if re, err := regexp.Compile(cfg.CustomRegex); err == nil && re.MatchString(text) {
			reasons = append(reasons, "custom_regex")
		}
	}

	// Repetition (same message N+ times from same author)
	if cfg.RepetitionThreshold > 0 {
		e.mu.Lock()
		list := e.recentByAuthor[authorID]
		count := 0
		for _, prev := range list {
			if strings.TrimSpace(prev) == strings.TrimSpace(text) {
				count++
			}
		}
		// append current to recent
		list = append(list, text)
		if len(list) > e.maxRecent {
			list = list[len(list)-e.maxRecent:]
		}
		e.recentByAuthor[authorID] = list
		e.mu.Unlock()
		if count+1 >= cfg.RepetitionThreshold {
			reasons = append(reasons, "repetition")
		}
	}

	return Result{
		IsSpam:    len(reasons) > 0,
		Reasons:   reasons,
		Message:   text,
		AuthorID:  authorID,
		MessageID: messageID,
	}
}

// ClearRecent clears repetition history (e.g. when changing video).
func (e *Engine) ClearRecent() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.recentByAuthor = make(map[string][]string)
}
