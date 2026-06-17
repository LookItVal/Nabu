package main

import (
	"testing"
	"time"
)

func TestMain_SuccessfullConnection(t *testing.T) {
	go main()
	time.Sleep(30 * time.Second)
}
