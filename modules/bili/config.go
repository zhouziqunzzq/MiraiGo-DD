package bili

type Config struct {
	Subscription         map[int64][]int64 `yaml:"subscription"`
	PollingInterval      uint              `yaml:"polling_interval"`
	DanmuForwardKeywords []string          `yaml:"danmu_forward_keywords"`
}
