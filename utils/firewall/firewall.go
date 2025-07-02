package firewall

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Firewall struct {
	LastError error
	headRule  *chainRule
	lastRule  *chainRule
	chainLock *sync.RWMutex
	ctx       context.Context
	// name -> rule
	nameRuleMap sync.Map // name -> rule

	// rate limit
	rateLimitRuleMap       []*Rule // rule
	rateLimitRuleMapLocker *sync.RWMutex
	rateLimitMapIP         sync.Map     // ip -> count (atomic.Int32)
	rateLimitTicker        *time.Ticker // statistical cycle (millisecond)
	// ipaddr
	ipaddrRuleMap sync.Map // ip -> rule
	// ipcidr
	ipcidrRuleMap       []*Rule // rule
	ipcidrRuleMapLocker *sync.RWMutex
	// ua
	uaRuleMap sync.Map // ua (regexp) -> rule
}

func NewFirewall(ctx context.Context) *Firewall {
	fw := &Firewall{
		headRule:               nil,
		lastRule:               nil,
		ctx:                    ctx,
		chainLock:              new(sync.RWMutex),
		rateLimitRuleMap:       make([]*Rule, 0),
		rateLimitRuleMapLocker: new(sync.RWMutex),
		ipcidrRuleMap:          make([]*Rule, 0),
		ipcidrRuleMapLocker:    new(sync.RWMutex),
	}
	// create rate limit ticker
	rateLimitTicker := time.NewTicker(time.Second * 1) // 1s
	fw.rateLimitTicker = rateLimitTicker

	fw.ReadRules()
	go fw.autoDeleteRule(ctx)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-rateLimitTicker.C:
				fw.rateLimitMapIP.Clear() // clear rate limit map
			}
		}
	}()
	return fw
}

func (f *Firewall) AddRule(rule *Rule, head ...bool) bool {
	if rule.Name == "" || rule.Rule == "" {
		return false
	}
	rule.compiledRuleArgs = new(sync.Map) // init compiledRuleArgs
	f.DeleteRule(rule.Name)               // delete old rule
	f.chainLock.Lock()
	defer f.chainLock.Unlock()
	newChain := &chainRule{Rule: rule, next: nil}
	switch rule.Type {
	case "ipaddr": // add to ipaddrRuleMap
		f.ipaddrRuleMap.Store(rule.Rule, rule)
	case "ipcidr": // add to ipcidrRuleMap
		f.ipcidrRuleMapLocker.Lock()
		_, parsedNet, err := net.ParseCIDR(rule.Rule)
		if err != nil {
			f.LastError = fmt.Errorf("invalid ipcidr rule: %s", rule.Rule)
			break
		}
		rule.parsedIPNet = parsedNet
		f.ipcidrRuleMap = append(f.ipcidrRuleMap, rule)
		f.ipcidrRuleMapLocker.Unlock()
	case "useragent": // add to uaRuleMap
		parsedUARegex, err := regexp.Compile(rule.Rule)
		if err != nil {
			f.LastError = fmt.Errorf("invalid useragent rule: %s", rule.Rule)
			break
		}
		rule.parsedUARegex = parsedUARegex
		f.uaRuleMap.Store(rule.parsedUARegex, rule)
	case "ratelimit": // add to rateLimitRuleMap
		f.rateLimitRuleMapLocker.Lock()
		f.rateLimitRuleMap = append(f.rateLimitRuleMap, rule)
		f.rateLimitRuleMapLocker.Unlock()
	}

	// get args
	for _, arg := range rule.Args {
		argsArr := strings.Split(arg, "=")
		if len(argsArr) != 2 {
			continue
		}
		rule.compiledRuleArgs.Store(argsArr[0], argsArr[1])
		switch argsArr[0] {
		case "cycle":
			i, err := strconv.ParseInt(argsArr[1], 10, 64)
			if err != nil {
				f.LastError = fmt.Errorf("invalid cycle arg: %s", argsArr[1])
				break
			}
			f.rateLimitTicker.Reset(time.Second * time.Duration(i))
		}
	}

	f.nameRuleMap.Store(rule.Name, rule)

	if f.headRule == nil {
		f.headRule = newChain
		f.lastRule = newChain
	} else {
		if head != nil && head[0] { // add to head if head
			newChain.next = f.headRule
			f.headRule = newChain
		} else { // add to tail if none or not head
			f.lastRule.next = newChain
			f.lastRule = newChain
		}
	}
	return true
}

func (f *Firewall) DeleteRule(ruleName string) bool {
	f.chainLock.Lock()
	defer f.chainLock.Unlock()
	if f.headRule == nil {
		f.LastError = fmt.Errorf("no rule found: %s", ruleName)
		return false
	}
	if _, ok := f.nameRuleMap.Load(ruleName); !ok {
		f.LastError = fmt.Errorf("no rule found: %s", ruleName)
		return false
	}
	dummy := &chainRule{next: f.headRule} // dummy head
	prev := dummy

	// find rule
	for curr := dummy.next; curr != nil; curr = curr.next {
		if curr.Rule.Name == ruleName { // find rule
			switch curr.Rule.Type {
			case "ipaddr":
				f.ipaddrRuleMap.Delete(curr.Rule.Rule)
			case "ipcidr":
				f.ipcidrRuleMapLocker.Lock()
				for i, v := range f.ipcidrRuleMap {
					if v == curr.Rule {
						f.ipcidrRuleMap = append(f.ipcidrRuleMap[:i], f.ipcidrRuleMap[i+1:]...)
						break
					}
				}
				f.ipcidrRuleMapLocker.Unlock()
			case "useragent":
				var deleteKey any
				f.uaRuleMap.Range(func(key, value any) bool {
					RruleName := value.(*Rule).Name
					if RruleName == ruleName {
						deleteKey = key
						return false
					}
					return true
				})
				if deleteKey != nil {
					f.uaRuleMap.Delete(deleteKey)
				}
			case "ratelimit":
				f.rateLimitRuleMapLocker.Lock()
				for i, v := range f.rateLimitRuleMap {
					if v == curr.Rule {
						f.rateLimitRuleMap = append(f.rateLimitRuleMap[:i], f.rateLimitRuleMap[i+1:]...)
						break
					}
				}
				f.rateLimitRuleMapLocker.Unlock()
			}
			f.nameRuleMap.Delete(curr.Rule.Name)

			prev.next = curr.next   // set prev next to curr next( delete curr )
			if curr == f.lastRule { // if delete last rule
				f.lastRule = prev // set lastRule to prev
			}
			f.headRule = dummy.next
			return true
		}
		prev = curr
	}
	return false
}

// return Action Code,0 -> not found
func (f *Firewall) MatchRule(ip string, r *http.Request) (action int, reason string) {
	// add ip to rate limit map
	counter, _ := f.rateLimitMapIP.LoadOrStore(ip, &atomic.Int32{})
	count := counter.(*atomic.Int32).Add(1)

	// check ip addr rules
	if rule, ok := f.ipaddrRuleMap.Load(ip); ok {
		rule := rule.(*Rule)
		if rule.Timeout > 0 && rule.Timeout < time.Now().Unix() { // timeout
			return 0, ""
		}
		return rule.Action, rule.Reason
	}

	// parse rule to net.ip
	ruleIP := net.ParseIP(ip)
	if ruleIP != nil {
		// check ip cidr rules
		f.ipcidrRuleMapLocker.RLock()
		for _, rule := range f.ipcidrRuleMap {
			if rule.Timeout > 0 && rule.Timeout < time.Now().Unix() { // timeout
				continue
			}
			var mask *net.IPNet
			if rule.parsedIPNet == nil { // has parsed ip net
				// parse ip cidr
				_, mask_, err := net.ParseCIDR(rule.Rule)
				if err != nil {
					continue
				}
				mask = mask_
			} else {
				mask = rule.parsedIPNet
			}

			// check ip(rule) in cidr
			if mask.Contains(ruleIP) {
				f.ipcidrRuleMapLocker.RUnlock()
				return rule.Action, rule.Reason
			}
		}
		f.ipcidrRuleMapLocker.RUnlock()
	}

	// check ua rules
	if r != nil {
		isMatched := false
		var selectedRule *Rule
		f.uaRuleMap.Range(func(key, value any) bool {
			regex := key.(*regexp.Regexp)
			rule := value.(*Rule)
			if regex.MatchString(r.Header.Get("User-Agent")) { // match ua
				isMatched = true
				if rule.Timeout > 0 && rule.Timeout < time.Now().Unix() { // timeout
					return true
				}
				selectedRule = rule
				return false
			}
			return true
		})
		if isMatched {
			return selectedRule.Action, selectedRule.Reason
		}
	}

	// check rate limit
	f.rateLimitRuleMapLocker.RLock()
	for _, rule := range f.rateLimitRuleMap {
		if rule.Timeout > 0 && rule.Timeout < time.Now().Unix() { // timeout
			f.rateLimitRuleMapLocker.RUnlock()
			return 0, ""
		}
		blockTime := time.Now().Unix() + 60               // 60s
		v, ok := rule.compiledRuleArgs.Load("block_time") // set block time
		if ok {
			vStr := v.(string)
			vInt, err := strconv.ParseInt(vStr, 10, 64)
			if err == nil {
				blockTime = time.Now().Unix() + vInt
			}
		}
		// check count
		ruleCount, err := strconv.Atoi(rule.Rule)
		if err != nil {
			continue
		}
		if count >= int32(ruleCount) { // block
			f.AddRule(&Rule{
				Name:    "rate_limited_ip_" + ip,
				Action:  1,
				Rule:    ip,
				Type:    "ipaddr",
				Reason:  rule.Reason,
				Timeout: blockTime,
			}, true) // add block rule to head
			f.rateLimitRuleMapLocker.RUnlock()
			return rule.Action, rule.Reason
		}
	}

	return 0, ""
}

func (f *Firewall) ShowRules() []*Rule {
	rules := make([]*Rule, 0)
	for curr := f.headRule; curr != nil; curr = curr.next { // keep the order
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
			var toDelete []string

			// collect timeout rules
			f.nameRuleMap.Range(func(key, value interface{}) bool {
				rule := value.(*Rule)
				if rule.Timeout > 0 && rule.Timeout < time.Now().Unix() {
					toDelete = append(toDelete, rule.Name)
				}
				return true
			})

			// 批量删除
			for _, name := range toDelete {
				f.DeleteRule(name)
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
		f.LastError = fmt.Errorf("read firewall config file failed: %s", err)
		return false
	}
	// parse json
	type Config struct {
		Rules []Rule `json:"rules"`
	}
	var cfg Config
	err = json.Unmarshal(cfg_filebin, &cfg)
	if err != nil {
		f.LastError = fmt.Errorf("invalid firewall config: %s", err)
		return false
	}
	// add rules
	for _, rule := range cfg.Rules {
		f.AddRule(&rule)
	}
	return true
}

func (f *Firewall) SaveRules() bool {
	const cfgPath = "configs/firewall.json"

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
		cfg.Rules = append(cfg.Rules, *r)
	}

	// 生成带格式的 JSON
	data, err := json.MarshalIndent(cfg, "", "    ")
	if err != nil {
		f.LastError = fmt.Errorf("marshal firewall config failed: %s", err)
		return false
	}

	// 写入文件
	if err := os.WriteFile(cfgPath, data, 0644); err != nil {
		f.LastError = fmt.Errorf("write firewall config file failed: %s", err)
		return false
	}

	return true
}

type chainRule struct {
	next *chainRule
	Rule *Rule
}

type Rule struct {
	Name    string   `json:"name"`    // rule name
	Action  int      `json:"action"`  // 1: block, 0: allow/default
	Rule    string   `json:"rule"`    // here store the ip addr
	Type    string   `json:"type"`    // "ipaddr": ip address, "ipcidr": ip address with mask
	Timeout int64    `json:"timeout"` // timeout timestamp
	Reason  string   `json:"reason"`  // reason for block or allow
	Args    []string `json:"args"`    // extra args for rule

	parsedIPNet      *net.IPNet     // parsed ip net for ipcidr rule
	parsedUARegex    *regexp.Regexp // parsed ua regex for useragent rule
	compiledRuleArgs *sync.Map      // compiled args for rules
}
