package nicrudns

import (
	"bytes"
	"encoding/json"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
	"os"
)

func (client *Client) UpdateCacheFile(token *oauth2.Token) error {
	tokenPath := client.config.CachePath
	file, err := os.Create(tokenPath)
	if err != nil {
		return errors.Wrap(err, CreateFileError.Error())
	}
	if err := json.NewEncoder(file).Encode(token); err != nil {
		return errors.Wrap(err, JsonEncodeError.Error())
	}
	return nil
}

func (client *Client) ReadCacheFile() (*oauth2.Token, error) {
	tokenPath := client.config.CachePath
	data, err := os.ReadFile(tokenPath)
	if err != nil {
		return nil, errors.Wrap(err, ReadFileError.Error())
	}
	token := &oauth2.Token{}
	if err := json.NewDecoder(bytes.NewBuffer(data)).Decode(&token); err != nil {
		return nil, errors.Wrap(err, JsonDecodeError.Error())
	} else {
		return token, nil
	}
}
