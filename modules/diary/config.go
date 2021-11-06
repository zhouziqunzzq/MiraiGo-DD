package diary

type Config struct {
	EnabledGroups []int64 `yaml:"enabled_groups"`
	RedisAddr     string  `yaml:"redis_addr"`
	RedisPassword string  `yaml:"redis_password"`
	RedisDb       int     `yaml:"redis_db"`
}
