package main

import (
	"context"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/yogabagas/business-idea-mvp/internal/agentflow/clickup"
	"github.com/yogabagas/business-idea-mvp/internal/agentflow/config"
	"github.com/yogabagas/business-idea-mvp/internal/agentflow/pipeline"
)

func main() {
	_ = godotenv.Load()
	cfg := config.LoadAgentFlow()
	r := gin.Default()

	// Health for debugging
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// ClickUp webhook: expect taskStatusUpdated with status = "in progress" (or AGENTFLOW_IN_PROGRESS_STATUS)
	r.POST("/webhook/clickup", func(c *gin.Context) {
		var payload clickup.WebhookPayload
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
			return
		}
		if !payload.IsTaskStatusUpdatedTo(cfg.InProgressStatus) {
			c.JSON(http.StatusOK, gin.H{"ok": true, "skipped": "status not in progress"})
			return
		}
		taskID := payload.TaskID
		if taskID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing task_id"})
			return
		}
		// Run pipeline asynchronously so we respond to ClickUp quickly (use Background so it continues after response)
		go func() {
			_, err := pipeline.Run(context.Background(), cfg, taskID)
			if err != nil {
				log.Printf("pipeline error for task %s: %v", taskID, err)
			}
		}()
		c.JSON(http.StatusOK, gin.H{"ok": true, "task_id": taskID})
	})

	log.Printf("ClickUp Agent Flow listening on :%s", cfg.AgentFlowPort)
	if err := r.Run(":" + cfg.AgentFlowPort); err != nil {
		log.Fatal(err)
	}
}
