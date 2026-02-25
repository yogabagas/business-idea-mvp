package api

import (
	"github.com/gin-gonic/gin"
)

func (h *Handlers) Register(r *gin.Engine) {
	r.Use(corsMiddleware(h.Cfg.FrontendOrigin))

	r.GET("/api/auth/login", h.Login)
	r.GET("/api/auth/callback", h.Callback)
	r.POST("/api/auth/logout", h.Logout)

	// Optional auth (for /me and for protected routes)
	apiGroup := r.Group("/api")
	apiGroup.Use(h.OptionalAuth)
	{
		apiGroup.GET("/auth/me", h.Me)
		apiGroup.GET("/live-chat/id", h.RequireAuth, h.GetLiveChatID)
		apiGroup.POST("/live-chat/messages", h.RequireAuth, h.GetMessages)
	}
}

func corsMiddleware(origin string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}
