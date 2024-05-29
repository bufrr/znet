package main

import (
	"github.com/bufrr/znet/examples"
	"time"
)

func main() {
	s := examples.StartSeed()
	time.Sleep(3 * time.Second)

	examples.StartCluster(s)

	select {}
}
