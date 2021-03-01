package naive_chatbot

type Config struct {
	EnabledGroups     []int64 `yaml:"enabled_groups"`
	NumPrediction     int64   `yaml:"n_prediction"`
	TimeOffsetSeconds int64   `yaml:"time_offset_seconds"`
	SimCutoff         float32 `yaml:"sim_cutoff"`
	GrpcServerAddr    string  `yaml:"grpc_server_addr"`
	TriggerProb       float32 `yaml:"trigger_prob"`
}
