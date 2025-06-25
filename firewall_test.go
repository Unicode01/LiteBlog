package main_test

import (
	"LiteBlog/utils/firewall"
	"context"
	"fmt"
	"testing"
	"time"
)

func Test_firewall(t *testing.T) {
	wall := firewall.NewFirewall(context.Background())
	for range 1000000 {
		time.Sleep(time.Millisecond)
		i, r := wall.MatchRule("127.0.0.1", nil)
		fmt.Printf("%d,reason:%s\n", i, r)
	}
}
