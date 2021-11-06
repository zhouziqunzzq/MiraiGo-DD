package diary

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/zhouziqunzzq/MiraiGo-DD/bot"
	"github.com/zhouziqunzzq/MiraiGo-DD/config"
	"github.com/zhouziqunzzq/MiraiGo-DD/utils"
	"gopkg.in/yaml.v2"
	"sync"
	"time"
)

const (
	TickDownPeriod        = time.Hour * 24 // one day
	TickDownCheckInterval = time.Second * 60
)

type diary struct {
	isEnabled     bool
	config        Config
	enabledGroups map[int64]bool

	rdb            *redis.Client
	redisCtx       context.Context
	redisCtxCancel context.CancelFunc

	workerWg        sync.WaitGroup
	workerCtx       context.Context
	workerCtxCancel context.CancelFunc
	initFinish      chan bool

	attributes     map[string]*Attribute // "groupId-userId" -> *Attribute
	attributesRwMu sync.RWMutex
}

func NewDiary() *diary {
	return &diary{
		isEnabled:     false,
		config:        Config{},
		enabledGroups: make(map[int64]bool),
		initFinish:    make(chan bool),
		attributes:    make(map[string]*Attribute),
	}
}

func (m *diary) MiraiGoModule() bot.ModuleInfo {
	return bot.ModuleInfo{
		ID:       ModuleName,
		Instance: instance,
	}
}

func (m *diary) Init() {
	// check is_enabled
	m.isEnabled = config.GlobalConfig.GetBool("modules." + ModuleName + ".is_enabled")
	if !m.isEnabled {
		logger.Info("this module is disabled by global config")
		return
	}

	// load module config
	configPath := config.GlobalConfig.GetString("modules." + ModuleName + ".config_path")
	if configPath == "" {
		configPath = "./diary.yaml"
	}
	logger.Debugf("reading config from %s", configPath)
	cb := utils.ReadFile(configPath)
	err := yaml.Unmarshal(cb, &m.config)
	if err != nil {
		logger.WithError(err).Errorf("unable to read config file in %s", configPath)
		m.isEnabled = false
		return
	}

	// init redis cli
	m.rdb = redis.NewClient(&redis.Options{
		Addr:     m.config.RedisAddr,
		Password: m.config.RedisPassword,
		DB:       m.config.RedisDb,
	})

	// init contexts
	m.redisCtx, m.redisCtxCancel = context.WithCancel(context.Background())
	m.workerCtx, m.workerCtxCancel = context.WithCancel(context.Background())

	// load enabled groups
	for _, groupCode := range m.config.EnabledGroups {
		m.enabledGroups[groupCode] = true
		logger.Infof("diary enabled for group %d", groupCode)
	}
}

func (m *diary) PostInit() {}

func (m *diary) Serve(b *bot.Bot) {
	if m.isEnabled {
		m.registerCallbacks(b)
	}
}

func (m *diary) Start(b *bot.Bot) {
	if !m.isEnabled {
		return
	}

	// start local attributes init goroutine
	go func() {
		m.workerWg.Add(1)
		defer m.workerWg.Done()
		m.initLocalAttributesCache()
	}()

	// start ttl tick-down goroutine
	go func() {
		m.workerWg.Add(1)
		defer m.workerWg.Done()
		m.tickDownTtlMainLoop()
	}()
}

func (m *diary) Stop(b *bot.Bot, wg *sync.WaitGroup) {
	defer wg.Done()

	if !m.isEnabled {
		return
	}

	// sync all Attributes back to redis
	if err := m.syncAttributeAll(); err != nil {
		logger.WithError(err).Errorf("failed to sync all Attributes back to redis")
	}

	// cancel all redis queries
	m.redisCtxCancel()

	// stop all workers
	m.workerCtxCancel()
	m.workerWg.Wait()
}

//func (m *diary) handleGroupMessage(qqClient *client.QQClient, groupMessage *message.GroupMessage) {
//	//
//}

func (m *diary) registerCallbacks(b *bot.Bot) {
	// b.OnGroupMessage(m.handleGroupMessage)
}

func GetId(groupId, userId int64) string {
	return fmt.Sprintf("%d-%d", groupId, userId)
}

func GetGroupIdRegex(groupId int64) string {
	return fmt.Sprintf("%d-*", groupId)
}

// getAttribute queries redis for Attribute identified by id
// and updates local cache.
func (m *diary) getAttribute(id string) (*Attribute, error) {
	rs, err := m.rdb.Get(m.redisCtx, id).Result()
	if err != nil {
		return nil, err
	}

	var rst Attribute
	if err = json.Unmarshal([]byte(rs), &rst); err != nil {
		return nil, err
	}

	m.attributesRwMu.Lock()
	defer m.attributesRwMu.Unlock()
	m.attributes[id] = &rst
	return &rst, nil
}

// getAttributeBatch queries redis for Attribute identified by a list of ids
// and updates local cache
func (m *diary) getAttributeBatch(ids []string) ([]*Attribute, error) {
	rs, err := m.rdb.MGet(m.redisCtx, ids...).Result()
	if err != nil {
		return nil, err
	}

	attrs := make([]*Attribute, len(ids))
	for i, rss := range rs {
		if rsJson, ok := rss.(string); !ok {
			return nil, errors.New(fmt.Sprintf("redis returns non-json string result: %v", rss))
		} else {
			attrs[i] = new(Attribute)
			if err = json.Unmarshal([]byte(rsJson), attrs[i]); err != nil {
				return nil, err
			}
		}
	}

	m.attributesRwMu.Lock()
	defer m.attributesRwMu.Unlock()
	for i, id := range ids {
		m.attributes[id] = attrs[i]
	}
	return attrs, nil
}

// syncAttribute syncs Attribute in local cache identified by id
// back to redis.
func (m *diary) syncAttribute(id string) error {
	m.attributesRwMu.RLock()
	defer m.attributesRwMu.RUnlock()
	if attr, ok := m.attributes[id]; ok {
		attrJson, err := json.Marshal(*attr)
		if err != nil {
			return err
		}
		if err = m.rdb.Set(m.redisCtx, id, string(attrJson), 0).Err(); err != nil {
			return err
		}
		return nil
	}

	return errors.New(fmt.Sprintf("invalid id=%s", id))
}

func (m *diary) syncAttributeAll() (err error) {
	m.attributesRwMu.RLock()
	defer m.attributesRwMu.RUnlock()
	for id, _ := range m.attributes {
		m.attributesRwMu.RUnlock()
		if e := m.syncAttribute(id); e != nil {
			err = e
		}
		m.attributesRwMu.RLock()
	}
	return
}

// initLocalAttributesCache fetches Attributes for all users
// in each of the groups with diary module enabled.
func (m *diary) initLocalAttributesCache() {
	defer func() {
		m.initFinish <- true
	}()

	for groupId, _ := range m.enabledGroups {
		// search for all keys in redis matching "groupId-*"
		ids, err := m.rdb.Keys(m.redisCtx, GetGroupIdRegex(groupId)).Result()
		switch {
		case err == redis.Nil:
			logger.Infof("no user records found for groupId=%d, skipping", groupId)
		case err != nil:
			logger.WithError(err).
				Errorf("failed to query redis in initLocalAttributesCache with groupId=%d", groupId)
		case len(ids) == 0:
			continue
		default:
			// no err and ids is not empty
			if attrs, err := m.getAttributeBatch(ids); err != nil {
				logger.WithError(err).
					Errorf("failed to query redis in initLocalAttributesCache with ids=%v", ids)
			} else {
				for i, id := range ids {
					logger.Infof("initialize Attribute for id=%s with values %v", id, *attrs[i])
				}
			}
		}

		// cancellation point
		select {
		case <-m.workerCtx.Done():
			return
		default:
			continue
		}
	}
}

// tickDownTtlMainLoop checks TtlTs for every user every TickDownCheckInterval,
// and decrement ttl for corresponding user by (now() - TtlTs) / TickDownPeriod
// and updates TtlTs to now().
func (m *diary) tickDownTtlMainLoop() {
	// wait for init to finish
	// cancellation point
	select {
	case <-m.initFinish:
		break
	case <-m.workerCtx.Done():
		return
	}

	ticker := time.NewTicker(TickDownCheckInterval)
	for {
		now := time.Now()

		m.attributesRwMu.RLock()
		for id, attr := range m.attributes {
			elapsed := now.Sub(time.Unix(attr.TtlTs, 0))
			if elapsed >= TickDownPeriod {
				// need to tick down Ttl and update TtlTs
				m.attributesRwMu.RUnlock()

				m.attributesRwMu.Lock()
				attr.Ttl -= int64(elapsed / TickDownPeriod)
				attr.TtlTs = now.Unix()
				m.attributesRwMu.Unlock()
				logger.Infof("tick-down Ttl for id=%s, now Attribute: %v", id, *attr)

				// TODO: send group message to notify the user about updated Ttl

				if err := m.syncAttribute(id); err != nil {
					logger.WithError(err).
						Errorf("failed to sync Attribute with id=%s back to redis", id)
				}

				m.attributesRwMu.RLock()
			}
		}
		m.attributesRwMu.RUnlock()

		// cancellation point
		select {
		case <-ticker.C:
			continue
		case <-m.workerCtx.Done():
			ticker.Stop()
			return
		}
	}
}
