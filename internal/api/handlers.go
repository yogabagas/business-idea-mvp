package api

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yogabagas/business-idea-mvp/internal/auth"
	"github.com/yogabagas/business-idea-mvp/internal/config"
	"github.com/yogabagas/business-idea-mvp/internal/filters"
	"github.com/yogabagas/business-idea-mvp/internal/store"
	"github.com/yogabagas/business-idea-mvp/internal/youtube"
)

const sessionCookieName = "session_id"

type Handlers struct {
	Cfg          *config.Config
	GoogleAuth   *auth.GoogleAuth
	FilterEngine *filters.Engine
}

func NewHandlers(cfg *config.Config, ga *auth.GoogleAuth, fe *filters.Engine) *Handlers {
	return &Handlers{Cfg: cfg, GoogleAuth: ga, FilterEngine: fe}
}

func (h *Handlers) ytClient(c *gin.Context) (*youtube.Client, error) {
	token := auth.GetTokenFromContext(c)
	if token == nil {
		return nil, nil
	}
	svc, err := auth.YouTubeService(c.Request.Context(), token)
	if err != nil {
		return nil, err
	}
	return youtube.NewClient(svc), nil
}

// OptionalAuth sets oauth token in context if session cookie is valid. Does not abort.
func (h *Handlers) OptionalAuth(c *gin.Context) {
	sessionID, _ := c.Cookie(sessionCookieName)
	if sessionID != "" {
		if token := store.GetToken(sessionID); token != nil {
			c.Set("oauth_token", token)
		}
	}
	c.Next()
}

// RequireAuth middleware: expects session cookie and sets oauth token in context.
func (h *Handlers) RequireAuth(c *gin.Context) {
	if auth.GetTokenFromContext(c) == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not logged in"})
		c.Abort()
		return
	}
	c.Next()
}

// Login redirects to Google OAuth.
func (h *Handlers) Login(c *gin.Context) {
	state, err := auth.GenerateState()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate state"})
		return
	}
	c.SetCookie("oauth_state", state, 600, "/", "", false, false)
	c.Redirect(http.StatusFound, h.GoogleAuth.AuthCodeURL(state))
}

// Callback handles OAuth callback, creates session, redirects to frontend.
func (h *Handlers) Callback(c *gin.Context) {
	state, _ := c.Cookie("oauth_state")
	if state == "" || state != c.Query("state") {
		c.Redirect(http.StatusFound, h.Cfg.FrontendOrigin+"/?error=invalid_state")
		return
	}
	c.SetCookie("oauth_state", "", -1, "/", "", false, false)
	code := c.Query("code")
	if code == "" {
		c.Redirect(http.StatusFound, h.Cfg.FrontendOrigin+"/?error=no_code")
		return
	}
	token, err := h.GoogleAuth.Exchange(c.Request.Context(), code)
	if err != nil {
		c.Redirect(http.StatusFound, h.Cfg.FrontendOrigin+"/?error=exchange_failed")
		return
	}
	sessionID := store.NewSession(token)
	c.SetCookie(sessionCookieName, sessionID, 24*3600, "/", "", false, true)
	c.Redirect(http.StatusFound, h.Cfg.FrontendOrigin+"/")
}

// Logout clears session.
func (h *Handlers) Logout(c *gin.Context) {
	sessionID, _ := c.Cookie(sessionCookieName)
	if sessionID != "" {
		store.DeleteSession(sessionID)
	}
	c.SetCookie(sessionCookieName, "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// Me returns auth status (and optional user info from YouTube).
func (h *Handlers) Me(c *gin.Context) {
	token := auth.GetTokenFromContext(c)
	if token == nil {
		c.JSON(http.StatusOK, gin.H{"loggedIn": false})
		return
	}
	c.JSON(http.StatusOK, gin.H{"loggedIn": true})
}

// GetLiveChatID returns liveChatId for a video ID.
func (h *Handlers) GetLiveChatID(c *gin.Context) {
	videoID := strings.TrimSpace(c.Query("videoId"))
	if videoID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "videoId required"})
		return
	}
	// Extract video ID from URL if needed
	if strings.Contains(videoID, "v=") {
		for _, part := range strings.Split(videoID, "&") {
			if strings.HasPrefix(part, "v=") {
				videoID = strings.TrimPrefix(part, "v=")
				break
			}
		}
	}
	if len(videoID) > 20 {
		videoID = videoID[:20]
	}
	yt, err := h.ytClient(c)
	if err != nil || yt == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	liveChatID, err := yt.GetLiveChatID(ctx, videoID)
	if err != nil {
		if err == youtube.ErrNoVideo || err == youtube.ErrNotLive {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	h.FilterEngine.ClearRecent()
	c.JSON(http.StatusOK, gin.H{"liveChatId": liveChatID, "videoId": videoID})
}

// ChatMessage is a single message with spam result for the frontend.
type ChatMessage struct {
	ID          string   `json:"id"`
	Text        string   `json:"text"`
	AuthorName  string   `json:"authorName"`
	AuthorID    string   `json:"authorId"`
	ProfileURL  string   `json:"profileImageUrl,omitempty"`
	PublishedAt string   `json:"publishedAt"`
	IsSpam      bool     `json:"isSpam"`
	Reasons     []string `json:"reasons,omitempty"`
}

// GetMessagesRequest is the body for POST /api/live-chat/messages.
type GetMessagesRequest struct {
	LiveChatID  string         `json:"liveChatId"`
	PageToken   string         `json:"pageToken"`
	FilterConfig *filters.Config `json:"filterConfig"`
}

// GetMessages returns latest live chat messages with spam filtering applied.
func (h *Handlers) GetMessages(c *gin.Context) {
	var req GetMessagesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if req.LiveChatID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "liveChatId required"})
		return
	}
	cfg := req.FilterConfig
	if cfg == nil {
		cfg = &filters.Config{}
	}
	yt, err := h.ytClient(c)
	if err != nil || yt == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()
	items, nextToken, pollMs, err := yt.ListMessages(ctx, req.LiveChatID, req.PageToken, 200)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make([]ChatMessage, 0, len(items))
	for _, m := range items {
		text := ""
		if m.Snippet != nil {
			text = m.Snippet.DisplayMessage
		}
		authorID := ""
		authorName := ""
		profileURL := ""
		if m.AuthorDetails != nil {
			authorID = m.AuthorDetails.ChannelId
			authorName = m.AuthorDetails.DisplayName
			profileURL = m.AuthorDetails.ProfileImageUrl
		}
		res := h.FilterEngine.Apply(cfg, m.Id, authorID, text)
		publishedAt := ""
		if m.Snippet != nil {
			publishedAt = m.Snippet.PublishedAt
		}
		msg := ChatMessage{
			ID:          m.Id,
			Text:        text,
			AuthorName:  authorName,
			AuthorID:    authorID,
			ProfileURL:  profileURL,
			PublishedAt: publishedAt,
			IsSpam:      res.IsSpam,
			Reasons:     res.Reasons,
		}
		if !cfg.HideFiltered || !res.IsSpam {
			out = append(out, msg)
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"messages":     out,
		"nextPageToken": nextToken,
		"pollingIntervalMillis": pollMs,
	})
}
