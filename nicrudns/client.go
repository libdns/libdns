package nicrudns

import (
	"github.com/pkg/errors"
	"net/http"
)

type Client struct {
	config       *Config
	oauth2client *http.Client
}

func NewClient(config *Config) IClient {
	return &Client{config: config}
}

func (client *Client) Do(r *http.Request) (*http.Response, error) {
	oauth2client, err := client.GetOauth2Client()
	if err != nil {
		return nil, errors.Wrap(err, Oauth2ClientError.Error())
	}
	return oauth2client.Do(r)
}
