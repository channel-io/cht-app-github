package config

type Stage = string

const (
	StageDevelopment Stage = "development"
	StageTest        Stage = "test"
	StageExp         Stage = "exp"
	StageProduction  Stage = "production"
)

var config *Config

func Get() *Config {
	return config
}

type Config struct {
	Stage Stage
	Build struct {
		Version string
		Commit  string
		Time    string
	}
	API struct {
		Public struct {
			HTTP struct {
				Port string
			}
		}
		Metric struct {
			HTTP struct {
				Port string
			}
		}
	}
	Log struct {
		Level string
	}

	Github struct {
		App struct {
			Id             int64
			ClientId       string
			WebhookSecret  string
			PrivateKeyPath string
		}
		Properties struct {
			ChannelIdKey      string
			GroupIdKey        string
			ReleaseGroupIdKey string
		}
	}

	ChannelTalk struct {
		DeskUrl           string
		BotName           string
		ManagerProfileKey string
		AppStore          struct {
			BaseUrl string
		}
		App struct {
			ID     string
			Secret string
		}
	}
}
