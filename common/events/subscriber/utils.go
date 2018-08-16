package subscriber

import (
	"encoding/base64"
	"encoding/json"

	"github.com/pkg/errors"
)

func DecodeJSONMessage(message *Message, dst interface{}) error {
	data, err := DecodeTextMessage(message)
	if err != nil {
		return errors.Wrap(err, "couldn't base64 decode message")
	}

	err = json.Unmarshal(data, dst)
	if err != nil {
		return errors.Wrap(err, "couldn't unmarshal send message event")
	}

	return nil
}

func DecodeTextMessage(message *Message) ([]byte, error) {
	return base64.StdEncoding.DecodeString(string(message.Data))
}
