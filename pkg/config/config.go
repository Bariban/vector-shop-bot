package config

import "github.com/spf13/viper"

type Messages struct {
	Responses
	Errors
}

type Responses struct {
	Start           string `mapstructure:"start"`
	UnknownCommand  string `mapstructure:"unknown_command"`
	LinkSaved       string `mapstructure:"link_saved"`
	SendPhoto       string `mapstructure:"send_photo"`
	KeyboardUpdated string `mapstructure:"keyboard_updated"`
}

type Errors struct {
	Default      string `mapstructure:"default"`
	InvalidURL   string `mapstructure:"invalid_url"`
	UnableToSave string `mapstructure:"unable_to_save"`
}

type Options struct {
	VectorExpectedDim    int    `mapstructure:"vector_expected_dim"`
	VectorEfConstruction int    `mapstructure:"vector_ef_construction"`
	VectorCommunication  int    `mapstructure:"vector_communication"`
	VectorIndexFile      string `mapstructure:"vector_index_file"`
}

type Config struct {
	TelegramToken     string `mapstructure:"telegram_token"`

	BotURL     string `mapstructure:"bot_url"`
	BoltDBFile string `mapstructure:"db_file"`

	Messages Messages
	Options  Options
}

func Init() (*Config, error) {
	if err := setUpViper(); err != nil {
		return nil, err
	}

	var cfg Config
	if err := unmarshal(&cfg); err != nil {
		return nil, err
	}

	if err := fromEnv(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func unmarshal(cfg *Config) error {
	if err := viper.Unmarshal(&cfg); err != nil {
		return err
	}

	if err := viper.UnmarshalKey("messages.response", &cfg.Messages.Responses); err != nil {
		return err
	}

	if err := viper.UnmarshalKey("messages.error", &cfg.Messages.Errors); err != nil {
		return err
	}

	return nil
}

func fromEnv(cfg *Config) error {
	if err := viper.BindEnv("token"); err != nil {
		return err
	}
	cfg.TelegramToken = viper.GetString("token")

	return nil
}

func setUpViper() error {
	viper.AddConfigPath("../../configs")
	viper.SetConfigName("main")

	return viper.ReadInConfig()
}
