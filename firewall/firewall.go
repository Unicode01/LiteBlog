package firewall

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"strconv"
	"sync"
	"time"
)

type Firewall struct {
	ruleLocker *sync.RWMutex
	headRule   *chainRule
	lastRule   *chainRule
	ctx        context.Context
	cancel     context.CancelFunc
}

func NewFirewall() *Firewall {
	ctx, cancle := context.WithCancel(context.Background())
	fw := &Firewall{ruleLocker: new(sync.RWMutex), headRule: nil, lastRule: nil, cancel: cancle, ctx: ctx}
	go fw.autoDeleteRule(ctx)
	fw.ReadRules()
	return fw
}

func (f *Firewall) AddRule(rule *Rule) bool {
	f.ruleLocker.Lock()
	defer f.ruleLocker.Unlock()
	newChain := &chainRule{Rule: rule, next: nil}
	if f.headRule == nil {
		f.headRule = newChain
		f.lastRule = newChain
	} else {
		f.lastRule.next = newChain
		f.lastRule = newChain
	}
	return true
}

func (f *Firewall) DeleteRule(rule string) bool {
	f.ruleLocker.Lock()
	defer f.ruleLocker.Unlock()
	if f.headRule == nil {
		return false
	}

	// if is head rule
	if f.headRule.Rule.Rule == rule {
		if f.headRule == f.lastRule {
			f.lastRule = nil
		}
		f.headRule = f.headRule.next
		return true
	}

	// find rule
	prev := f.headRule
	for curr := f.headRule.next; curr != nil; curr = curr.next {
		if curr.Rule.Rule == rule { // find rule
			prev.next = curr.next   // set prev next to curr next( delete curr )
			if curr == f.lastRule { // if delete last rule
				f.lastRule = prev // set lastRule to prev
			}
			return true
		}
		prev = curr
	}
	return false
}

// return Action Code,0 -> not found
func (f *Firewall) MatchRule(rule string) int {
	f.ruleLocker.RLock()
	defer f.ruleLocker.RUnlock()
	for curr := f.headRule; curr != nil; curr = curr.next {
		switch curr.Rule.Type {
		case "ipaddr":
			if curr.Rule.Rule == rule {
				if curr.Rule.Timeout > 0 && curr.Rule.Timeout < time.Now().Unix() { // timeout
					return 0
				}
				return curr.Rule.Action
			}
		case "ipcidr":
			// parse ip cidr
			_, mask, err := net.ParseCIDR(rule)
			if err != nil {
				continue
			}
			// parse rule to net.ip
			ruleIP := net.ParseIP(rule)
			if ruleIP == nil {
				continue
			}
			// check ip(rule) in cidr
			if mask.Contains(ruleIP) {
				if curr.Rule.Timeout > 0 && curr.Rule.Timeout < time.Now().Unix() { // timeout
					return 0
				}
				return curr.Rule.Action
			}
		}
	}
	return 0
}

func (f *Firewall) ShowRules() []*Rule {
	f.ruleLocker.RLock()
	defer f.ruleLocker.RUnlock()
	rules := make([]*Rule, 0)
	for curr := f.headRule; curr != nil; curr = curr.next {
		rules = append(rules, curr.Rule)
	}
	return rules
}

func (f *Firewall) autoDeleteRule(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			for curr := f.headRule; curr != nil; curr = curr.next {
				if curr.Rule.Timeout > 0 && curr.Rule.Timeout < time.Now().Unix() { // timeout
					f.DeleteRule(curr.Rule.Rule)
				}
			}
			f.SaveRules()
		}
		time.Sleep(time.Second * 5)
	}
}

// read rules from file,return true if success
func (f *Firewall) ReadRules() bool {
	cfg_path := "configs/firewall.json"
	// read from file
	cfg_filebin, err := os.ReadFile(cfg_path)
	if err != nil {
		return false
	}
	// parse json
	type Config struct {
		Rules []map[string]string `json:"rules"`
	}
	var cfg Config
	err = json.Unmarshal(cfg_filebin, &cfg)
	if err != nil {
		return false
	}
	// add rules
	for _, rule := range cfg.Rules {
		action, _ := strconv.Atoi(rule["action"])
		timeout, _ := strconv.ParseInt(rule["timeout"], 10, 64)
		f.AddRule(&Rule{Action: action, Rule: rule["rule"], Type: rule["type"], Timeout: timeout})
	}
	return true
}

func (f *Firewall) SaveRules() bool {
	const cfgPath = "configs/firewall.json"

	type Rule struct {
		Action  int    `json:"action"` // 保持原始数值类型
		Rule    string `json:"rule"`
		Type    string `json:"type"`
		Timeout int64  `json:"timeout"`
	}

	type Config struct {
		Rules []Rule `json:"rules"`
	}

	// 获取原始规则
	rawRules := f.ShowRules()

	// 转换规则格式
	cfg := Config{
		Rules: make([]Rule, 0, len(rawRules)),
	}
	for _, r := range rawRules {
		cfg.Rules = append(cfg.Rules, Rule{
			Action:  r.Action,
			Rule:    r.Rule,
			Timeout: r.Timeout,
		})
	}

	// 生成带格式的 JSON
	data, err := json.MarshalIndent(cfg, "", "\t")
	if err != nil {
		return false
	}

	// 写入文件
	if err := os.WriteFile(cfgPath, data, 0644); err != nil {
		return false
	}

	return true
}

type chainRule struct {
	next *chainRule
	Rule *Rule
}

type Rule struct {
	Action  int    // 1: block, 0: allow/default
	Rule    string // here store the ip addr
	Type    string // "ipaddr": ip address, "ipcidr": ip address with mask
	Timeout int64  // timeout timestamp
}
