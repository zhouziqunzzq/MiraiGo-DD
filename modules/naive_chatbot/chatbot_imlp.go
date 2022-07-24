package naive_chatbot

import (
	"context"
	"github.com/Mrs4s/MiraiGo/client"
	"github.com/Mrs4s/MiraiGo/message"
	"github.com/zhouziqunzzq/MiraiGo-DD/bot"
	"github.com/zhouziqunzzq/MiraiGo-DD/config"
	"github.com/zhouziqunzzq/MiraiGo-DD/modules/common"
	pb "github.com/zhouziqunzzq/MiraiGo-DD/modules/naive_chatbot/protos"
	"github.com/zhouziqunzzq/MiraiGo-DD/utils"
	"google.golang.org/grpc"
	"gopkg.in/yaml.v2"
	"math/rand"
	"sync"
	"time"
)

const (
	GrpcTimeout = 10 * time.Second
)

type chatbot struct {
	isEnabled bool
	config    Config
	//enabledGroupsMap map[int64]bool
	groupTriggerProb map[int64]float32
	conn             *grpc.ClientConn
	client           pb.ChatPredictorClient
}

func NewChatbot() *chatbot {
	return &chatbot{
		isEnabled:        false,
		config:           Config{},
		groupTriggerProb: make(map[int64]float32),
		conn:             nil,
		client:           nil,
	}
}

func (m *chatbot) MiraiGoModule() bot.ModuleInfo {
	return bot.ModuleInfo{
		ID:       ModuleName,
		Instance: instance,
	}
}

func (m *chatbot) Init() {
	// check is_enabled
	m.isEnabled = config.GlobalConfig.GetBool("modules." + ModuleName + ".is_enabled")
	if !m.isEnabled {
		logger.Info("this module is disabled by global config")
		return
	}

	// load module config
	configPath := config.GlobalConfig.GetString("modules." + ModuleName + ".config_path")
	if configPath == "" {
		configPath = "./naive_chatbot.yaml"
	}
	logger.Debugf("reading config from %s", configPath)
	cb := utils.ReadFile(configPath)
	err := yaml.Unmarshal(cb, &m.config)
	if err != nil {
		logger.WithError(err).Errorf("unable to read config file in %s", configPath)
		m.isEnabled = false
		return
	}
	// check trigger prob range
	if m.config.TriggerProb < 0.0 || m.config.TriggerProb > 1.0 {
		logger.WithError(err).Errorf("invalid trigger_prob %f provided in config", m.config.TriggerProb)
		m.isEnabled = false
		return
	}

	// load enabled groups
	for _, groupCode := range m.config.EnabledGroups {
		m.groupTriggerProb[groupCode] = m.config.TriggerProb
		logger.Infof(
			"naive chatbot enabled for group %d with default trigger prob %f",
			groupCode, m.config.TriggerProb,
		)
	}

	// connect to grpc server
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	m.conn, err = grpc.Dial(m.config.GrpcServerAddr, opts...)
	if err != nil {
		logger.WithError(err).Errorf("unable to connect to grpc server %s", m.config.GrpcServerAddr)
		m.isEnabled = false
		return
	}
	m.client = pb.NewChatPredictorClient(m.conn)
}

func (m *chatbot) PostInit() {}

func (m *chatbot) Serve(b *bot.Bot) {
	if m.isEnabled {
		m.registerCallbacks(b)
	}
}

func (m *chatbot) Start(b *bot.Bot) {}

func (m *chatbot) Stop(b *bot.Bot, wg *sync.WaitGroup) {
	defer wg.Done()

	if m.conn != nil {
		_ = m.conn.Close()
	}
}

func (m *chatbot) PredictOne(msg string) []*pb.PredictReply_PredictReplyElem {
	// prepare predict request
	req := &pb.PredictRequest{
		Msg:               &msg,
		NPrediction:       &m.config.NumPrediction,
		TimeOffsetSeconds: &m.config.TimeOffsetSeconds,
		SimCutoff:         &m.config.SimCutoff,
	}

	ctx, cancel := context.WithTimeout(context.Background(), GrpcTimeout)
	defer cancel()

	rsp, err := m.client.PredictOne(ctx, req)
	if err != nil {
		logger.WithError(err).Errorf(
			"failed to predict reply msg for chat \"%s\"",
			msg,
		)
		return nil
	} else {
		return rsp.Result
	}
}

func (m *chatbot) handleGroupMessage(qqClient *client.QQClient, groupMessage *message.GroupMessage) {
	// filter enabled groups
	triggerProb := float32(0.0)
	if tp, ok := m.groupTriggerProb[groupMessage.GroupCode]; !ok {
		logger.Debugf("ignoring group message from group chat %s(%d)",
			groupMessage.GroupName,
			groupMessage.GroupCode,
		)
		return
	} else {
		triggerProb = tp
	}

	chatReq := groupMessage.ToString()
	// ignore empty str and cmd
	if len(chatReq) == 0 || chatReq[0] == common.CmdIdentifier {
		return
	}

	// trigger with probability of trigger prob
	if rand.Float32() >= triggerProb {
		return
	}

	// call NaivePredictor over grpc
	rsp := m.PredictOne(chatReq)
	if rsp != nil && len(rsp) > 0 {
		idx := rand.Intn(len(rsp))
		chosenRsp := rsp[idx]
		msg := message.NewSendingMessage()
		msg.Append(message.NewText(*(chosenRsp.Msg)))
		qqClient.SendGroupMessage(groupMessage.GroupCode, msg)
		logger.Infof(
			"reply to group message \"%s\" from group %s(%d) with msg \"%s\" and sim %f",
			chatReq,
			groupMessage.GroupName,
			groupMessage.GroupCode,
			*(chosenRsp.Msg),
			*(chosenRsp.Sim),
		)
	}
}

func (m *chatbot) registerCallbacks(b *bot.Bot) {
	b.GroupMessageEvent.Subscribe(m.handleGroupMessage)
}
