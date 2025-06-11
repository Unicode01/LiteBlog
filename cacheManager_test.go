package main_test

import (
	lb "LiteBlog/utils"
	"bytes"
	"testing"
)

func Test_cacheManager(t *testing.T) {
	cm := lb.NewCacheManager(1000000, 100)
	t.Log("cacheManager test passed")
	reader := bytes.NewReader([]byte("test"))
	err := cm.AddCacheItem("test", reader, 1)
	if err != nil {
		t.Error(err)
	}
	reader = bytes.NewReader([]byte("test2"))
	err = cm.AddCacheItem("test", reader, 1) // replace cache item
	if err != nil {
		t.Error(err)
	}
	// time.Sleep(2 * time.Second)
	c, err := cm.GetCacheItem("test")
	if err != nil || c == nil {
		t.Error(err)
	}
	defer c.Close()
	buffer := make([]byte, 1024)
	n, err := c.Read(buffer)
	if err != nil {
		t.Error(err)
	}
	t.Log(string(buffer[:n]))
}
