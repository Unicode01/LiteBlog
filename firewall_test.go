package main_test

import (
	"LiteBlog/utils/firewall"
	"fmt"
	"testing"
	"time"
)

func Test_firewall(t *testing.T) {
	wall := firewall.NewFirewall()
	timeNow := time.Now().Nanosecond()
	for i := 0; i < 10000000; i++ {
		wall.AddRule(&firewall.Rule{
			Rule:   "myrule" + fmt.Sprintf("%d", i),
			Action: i,
			Type:   "ipaddr",
		})
	}
	fmt.Printf("Add Timeused: %d\n", time.Now().Nanosecond()-timeNow)

	timeNow = time.Now().Nanosecond()
	for i := 9000000; i < 9000010; i++ {
		fmt.Printf("match rule: %s, result: %d\n", "myrule"+fmt.Sprintf("%d", i), wall.MatchRule("myrule"+fmt.Sprintf("%d", i), nil))
	}
	fmt.Printf("Del Timeused: %d\n", time.Now().Nanosecond()-timeNow)

	timeNow = time.Now().Nanosecond()
	wall.ShowRules()
	fmt.Printf("Show Timeused: %d\n", time.Now().Nanosecond()-timeNow)
}
