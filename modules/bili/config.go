package bili

type Config struct {
	Subscription    map[int64][]int64 `yaml:"subscription"`
	PollingInterval uint              `yaml:"polling_interval"`
}
