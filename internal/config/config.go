package config

type Config struct {
	APIToken   string `yaml:"APIToken"`
	AppID      string `yaml:"AppID"`
	Secret     string `yaml:"Secret"`
	UserID     string `yaml:"UserID"`
	TemplateID string `yaml:"TemplateID"`
	BaseURL    string `yaml:"BaseURL"`
	DBPath     string `yaml:"DBPath"`
	Port       int    `yaml:"Port"`
}

func (c *Config) applyDefaults() {
	if c.DBPath == "" {
		c.DBPath = "data/wxpush.db"
	}
	if c.Port == 0 {
		c.Port = 8080
	}
}
