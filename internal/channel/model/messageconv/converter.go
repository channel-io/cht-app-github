package messageconv

import "github.com/channel-io/cht-app-github/internal/channel/model"

type Converter interface {
	Convert() model.Message
}
