package node

import (
	"fmt"
	"sync"

	"github.com/InazumaV/V2bX/api/panel"
	"github.com/InazumaV/V2bX/common/task"
	"github.com/InazumaV/V2bX/conf"
	vCore "github.com/InazumaV/V2bX/core"
	"github.com/InazumaV/V2bX/limiter"
	log "github.com/sirupsen/logrus"
)

type Controller struct {
	server    vCore.Core
	apiClient *panel.Client
	// tag 与 limiter 会被两个定时任务同时访问（拉配置的写、上报的读），
	// 原来直接裸读写属于数据竞争：极端情况下按旧 tag 上报，面板会直接丢弃。
	stateMu  sync.RWMutex
	tag      string
	limiter  *limiter.Limiter
	traffic  map[string]int64
	userList []panel.UserInfo
	aliveMap map[int]int
	info     *panel.NodeInfo
	// needRebuild 只由 nodeInfoMonitor 这个 goroutine 读写：
	// 重建节点失败后置位，下一轮强制用上一次收到的配置重试。
	needRebuild               bool
	nodeInfoMonitorPeriodic   *task.Task
	userReportPeriodic        *task.Task
	renewCertPeriodic         *task.Task
	dynamicSpeedLimitPeriodic *task.Task
	onlineIpReportPeriodic    *task.Task
	*conf.Options
}

func (c *Controller) Tag() string {
	c.stateMu.RLock()
	defer c.stateMu.RUnlock()
	return c.tag
}

func (c *Controller) setTag(tag string) {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()
	c.tag = tag
}

func (c *Controller) Limiter() *limiter.Limiter {
	c.stateMu.RLock()
	defer c.stateMu.RUnlock()
	return c.limiter
}

func (c *Controller) setLimiter(l *limiter.Limiter) {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()
	c.limiter = l
}

// NewController return a Node controller with default parameters.
func NewController(server vCore.Core, api *panel.Client, config *conf.Options) *Controller {
	controller := &Controller{
		server:    server,
		Options:   config,
		apiClient: api,
	}
	return controller
}

// Start implement the Start() function of the service interface
func (c *Controller) Start() error {
	// First fetch Node Info
	var err error
	node, err := c.apiClient.GetNodeInfo()
	if err != nil {
		return fmt.Errorf("get node info error: %s", err)
	}
	// Update user
	c.userList, err = c.apiClient.GetUserList()
	if err != nil {
		return fmt.Errorf("get user list error: %s", err)
	}
	if len(c.userList) == 0 {
		// 权限组里还没有用户（新建节点很常见）。以前直接返回错误，节点会反复重启；
		// 现在先空跑把节点注册上去，下一轮拉到用户会自动补。
		log.Warn("user list is empty, start with no user")
	}
	c.aliveMap, err = c.apiClient.GetUserAlive()
	if err != nil {
		// 面板偶发抖动不该让节点起不来：先用空表启动，下一轮拉取会补上。
		log.WithFields(log.Fields{
			"err": err,
		}).Warn("get alive list failed, start with empty alive list")
		c.aliveMap = make(map[int]int)
	}
	if len(c.Options.Name) == 0 {
		c.setTag(c.buildNodeTag(node))
	} else {
		c.setTag(c.Options.Name)
	}

	// add limiter
	l := limiter.AddLimiter(c.Tag(), &c.LimitConfig, c.userList, c.aliveMap)
	// add rule limiter
	if err = l.UpdateRule(&node.Rules); err != nil {
		return fmt.Errorf("update rule error: %s", err)
	}
	c.setLimiter(l)
	if node.Security == panel.Tls {
		err = c.requestCert()
		if err != nil {
			return fmt.Errorf("request cert error: %s", err)
		}
	}
	// Add new tag
	err = c.server.AddNode(c.Tag(), node, c.Options)
	if err != nil {
		return fmt.Errorf("add new node error: %s", err)
	}
	added := 0
	if len(c.userList) > 0 {
		added, err = c.server.AddUsers(&vCore.AddUsersParams{
			Tag:      c.Tag(),
			Users:    c.userList,
			NodeInfo: node,
		})
		if err != nil {
			return fmt.Errorf("add users error: %s", err)
		}
	}
	log.WithField("tag", c.Tag()).Infof("Added %d new users", added)
	c.info = node
	c.startTasks(node)
	return nil
}

// Close implement the Close() function of the service interface
func (c *Controller) Close() error {
	limiter.DeleteLimiter(c.Tag())
	if c.nodeInfoMonitorPeriodic != nil {
		c.nodeInfoMonitorPeriodic.Close()
	}
	if c.userReportPeriodic != nil {
		c.userReportPeriodic.Close()
	}
	if c.renewCertPeriodic != nil {
		c.renewCertPeriodic.Close()
	}
	if c.dynamicSpeedLimitPeriodic != nil {
		c.dynamicSpeedLimitPeriodic.Close()
	}
	if c.onlineIpReportPeriodic != nil {
		c.onlineIpReportPeriodic.Close()
	}
	err := c.server.DelNode(c.Tag())
	if err != nil {
		return fmt.Errorf("del node error: %s", err)
	}
	return nil
}

func (c *Controller) buildNodeTag(node *panel.NodeInfo) string {
	return fmt.Sprintf("[%s]-%s:%d", c.apiClient.APIHost, node.Type, node.Id)
}
