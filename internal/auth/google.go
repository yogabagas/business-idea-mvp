package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	ytv3 "google.golang.org/api/youtube/v3"
)

const (
	YouTubeReadOnlyScope = "https://www.googleapis.com/auth/youtube.readonly"
	stateCookieName      = "oauth_state"
	stateCookieMaxAge    = 600
)

type GoogleAuth struct {
	conf *oauth2.Config
}

func NewGoogleAuth(clientID, clientSecret, redirectURL string) *GoogleAuth {
	return &GoogleAuth{
		conf: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes:       []string{YouTubeReadOnlyScope},
			Endpoint:     google.Endpoint,
		},
	}
}

func (a *GoogleAuth) AuthCodeURL(state string) string {
	return a.conf.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

func (a *GoogleAuth) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	return a.conf.Exchange(ctx, code)
}

func (a *GoogleAuth) Client(ctx context.Context, t *oauth2.Token) *http.Client {
	return a.conf.Client(ctx, t)
}

func (a *GoogleAuth) TokenSource(ctx context.Context, t *oauth2.Token) oauth2.TokenSource {
	return a.conf.TokenSource(ctx, t)
}

func GenerateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func YouTubeService(ctx context.Context, token *oauth2.Token, opts ...option.ClientOption) (*ytv3.Service, error) {
	if token != nil {
		opts = append([]option.ClientOption{option.WithTokenSource(oauth2.StaticTokenSource(token))}, opts...)
	}
	return ytv3.NewService(ctx, opts...)
}

func GetTokenFromContext(c *gin.Context) *oauth2.Token {
	v, ok := c.Get("oauth_token")
	if !ok || v == nil {
		return nil
	}
	t, _ := v.(*oauth2.Token)
	return t
}
