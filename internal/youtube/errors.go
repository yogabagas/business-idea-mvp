package youtube

import "errors"

var (
	ErrNoVideo = errors.New("video not found")
	ErrNotLive = errors.New("video is not a live stream or has no active chat")
)
