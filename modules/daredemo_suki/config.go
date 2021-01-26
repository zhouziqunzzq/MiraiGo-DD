package daredemo_suki

type Config struct {
	EnabledGroups []int64  `yaml:"enabled_groups"`
	ImgPath       string   `yaml:"img_path"`
	Keywords      []string `yaml:"keywords"`
}
