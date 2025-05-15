package main_test

import (
	lb "LiteBlog"
	"context"
	"fmt"
	"testing"
)

var (
	count int = 1
)

func TestDeliver(t *testing.T) {
	t.Log("Testing deliver...")
	ctx := context.Background()
	dm := lb.NewDeliverManager(10000000, 1, ctx)
	// timestart := time.Now().UnixNano() / int64(time.Millisecond)
	for i := 0; i < 10000000; i++ {
		err := dm.AddTask(func() {
			count++
			// fmt.Printf("count: %d\n", count)
		})
		if err != nil {
			fmt.Println(err.Error())
		}
	}
	fmt.Printf("Task Added Done\n")
	select {}
}
