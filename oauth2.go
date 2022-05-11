package nicrudns

import (
	"context"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
	"net/http"
)

func (client *Client) GetOauth2Client() (*http.Client, error) {
	ctx := context.TODO()

	if client.oauth2client != nil {
		return client.oauth2client, nil
	}

	oauth2Config := oauth2.Config{
		ClientID:     client.config.Credentials.OAuth2ClientID,
		ClientSecret: client.config.Credentials.OAuth2SecretID,
		Endpoint: oauth2.Endpoint{
			TokenURL:  TokenURL,
			AuthStyle: oauth2.AuthStyleInParams,
		},
		Scopes: []string{OAuth2Scope},
	}

	cachedToken, _ := client.ReadCacheFile()
	if cachedToken != nil {
		client.oauth2client = oauth2Config.Client(ctx, cachedToken)
		_, err := client.GetServices()
		if err == nil {
			return client.oauth2client, nil
		}
	}

	oauth2Token, err := oauth2Config.PasswordCredentialsToken(ctx, client.config.Credentials.Username, client.config.Credentials.Password)
	if err != nil {
		return nil, errors.Wrap(err, AuthorizationError.Error())
	}

	client.oauth2client = oauth2Config.Client(ctx, oauth2Token)
	if err := client.UpdateCacheFile(oauth2Token); err != nil {
		return nil, errors.Wrap(err, UpdateTokenCacheFileError.Error())
	}

	return client.oauth2client, nil
}
