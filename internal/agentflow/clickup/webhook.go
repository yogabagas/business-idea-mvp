package clickup

import (
	"strings"
)

// WebhookPayload is the payload ClickUp sends on task events.
// See https://clickup.com/api/developer-portal/webhooktaskpayloads/
type WebhookPayload struct {
	Event        string        `json:"event"`
	TaskID       string        `json:"task_id"`
	WebhookID    string        `json:"webhook_id"`
	HistoryItems []HistoryItem `json:"history_items"`
}

type HistoryItem struct {
	ID       string      `json:"id"`
	Type     int         `json:"type"`
	Date     string      `json:"date"`
	Field    string      `json:"field"`
	ParentID string      `json:"parent_id"`
	User     *WebhookUser `json:"user"`
	Before   *StatusState `json:"before"`
	After    *StatusState `json:"after"`
}

type WebhookUser struct {
	ID        int    `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	Color     string `json:"color"`
	Initials  string `json:"initials"`
}

type StatusState struct {
	Status    string `json:"status"`
	Color     string `json:"color"`
	Type      string `json:"type"`
	OrderIndex int   `json:"orderindex"`
}

// IsTaskStatusUpdatedTo returns true if event is taskStatusUpdated and the new status (after) matches targetStatus (case-insensitive).
func (p *WebhookPayload) IsTaskStatusUpdatedTo(targetStatus string) bool {
	if p.Event != "taskStatusUpdated" || targetStatus == "" {
		return false
	}
	target := stringsToLowerTrim(targetStatus)
	for _, h := range p.HistoryItems {
		if h.Field != "status" || h.After == nil {
			continue
		}
		if stringsToLowerTrim(h.After.Status) == target {
			return true
		}
	}
	return false
}

func stringsToLowerTrim(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
