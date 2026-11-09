package configs

// NotificationConfig 保存通知渠道配置，默认关闭。
type NotificationConfig struct {
	AllowedRecipientDomains []string `yaml:"allowed_recipient_domains"`
	Enabled                 bool     `yaml:"enabled"`
	Host                    string   `yaml:"host"`
	Port                    int      `yaml:"port"`
	Username                string   `yaml:"username"`
	Password                string   `yaml:"password"`
	From                    string   `yaml:"from"`
	TLSMode                 string   `yaml:"tls_mode"`
	TimeoutSeconds          int      `yaml:"timeout_seconds"`
}

// GetAllowedRecipientDomains 返回当前允许的接收邮箱域名，未配置时仅允许 QQ 邮箱。
func GetAllowedRecipientDomains() []string {
	if c == nil || c.Notification == nil || len(c.Notification.AllowedRecipientDomains) == 0 {
		return []string{"qq.com"}
	}
	return c.Notification.AllowedRecipientDomains
}
