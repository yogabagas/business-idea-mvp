package youtube

import (
	"context"

	ytv3 "google.golang.org/api/youtube/v3"
)

type Client struct {
	service *ytv3.Service
}

func NewClient(service *ytv3.Service) *Client {
	return &Client{service: service}
}

// GetLiveChatID returns the active live chat ID for a video (live stream).
// Uses videos.list with part=liveStreamingDetails.
func (c *Client) GetLiveChatID(ctx context.Context, videoID string) (string, error) {
	call := c.service.Videos.List([]string{"liveStreamingDetails"}).Id(videoID)
	resp, err := call.Context(ctx).Do()
	if err != nil {
		return "", err
	}
	if len(resp.Items) == 0 {
		return "", ErrNoVideo
	}
	details := resp.Items[0].LiveStreamingDetails
	if details == nil || details.ActiveLiveChatId == "" {
		return "", ErrNotLive
	}
	return details.ActiveLiveChatId, nil
}

// ListMessages fetches live chat messages. Pass empty pageToken for first request.
// Returns messages, nextPageToken, pollingIntervalMillis, and any error.
func (c *Client) ListMessages(ctx context.Context, liveChatID, pageToken string, maxResults int) ([]*ytv3.LiveChatMessage, string, uint64, error) {
	if maxResults <= 0 {
		maxResults = 200
	}
	call := c.service.LiveChatMessages.List(liveChatID, []string{"snippet", "authorDetails"})
	call.MaxResults(int64(maxResults))
	if pageToken != "" {
		call.PageToken(pageToken)
	}
	resp, err := call.Context(ctx).Do()
	if err != nil {
		return nil, "", 0, err
	}
	return resp.Items, resp.NextPageToken, uint64(resp.PollingIntervalMillis), nil
}
