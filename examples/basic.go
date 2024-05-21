package main

import (
	"fmt"
	"github.com/bufrr/znet/config"
	"github.com/bufrr/znet/dht"
	"github.com/bufrr/znet/utils"
	"github.com/bufrr/znet/znode"
	"golang.org/x/crypto/sha3"
	"log"
	"time"
)

const seed = "http://127.0.0.1:12345/rpc12345"

func main() {
	s := seedStart()
	time.Sleep(3 * time.Second)

	p2pPort := 33333
	wsPort := 23333
	rpcPort := 13333
	sl := make([]string, 0)
	sl = append(sl, seed)
	znets := make([]*znode.Znode, 0)

	for i := 0; i < 10; i++ {
		h := sha3.New256().Sum([]byte("Hello" + string(rune(i))))
		keypair, _ := dht.GenerateKeyPair(h[:32])

		p2p := p2pPort + i
		ws := wsPort + i
		rpc := rpcPort + i

		c := config.Config{
			Transport: "tcp",
			P2pPort:   uint16(p2p),
			Keypair:   keypair,
			WsPort:    uint16(ws),
			RpcPort:   uint16(rpc),
			UdpPort:   config.DefaultUdpPort,
			VlcAddr:   "127.0.0.1:8050",
			SeedList:  sl,
		}

		ip, err := utils.GetExtIp(c.SeedList[0])
		if err != nil {
			log.Fatal("get ext ip err:", err)
		}
		c.Domain = ip

		znd, err := znode.NewZnode(c)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("port: %d, wsport: %d, id: %x\n", p2p, ws, znd.Nnet.GetLocalNode().Id)

		znd.ApplyBytesReceived()
		znd.ApplyNeighborAdded()
		znd.ApplyNeighborRemoved()

		znets = append(znets, znd)
	}

	for i := 0; i < len(znets); i++ {
		time.Sleep(112358 * time.Microsecond)

		err := znets[i].Start(false)
		if err != nil {
			log.Fatal(err)
			return
		}

		err = znets[i].Nnet.Join(s.Nnet.GetLocalNode().Addr)
		if err != nil {
			log.Fatal(err)
			return
		}
	}

	select {}
}

func seedStart() *znode.Znode {
	h := sha3.New256().Sum([]byte("Hello" + string(rune(100))))
	keypair, _ := dht.GenerateKeyPair(h[:32])
	c := config.Config{
		Transport: "tcp",
		P2pPort:   uint16(12344),
		Keypair:   keypair,
		WsPort:    uint16(12346),
		RpcPort:   uint16(12345),
		UdpPort:   config.DefaultUdpPort,
		VlcAddr:   "127.0.0.1:8050",
		Domain:    "127.0.0.1",
	}

	znd, err := znode.NewZnode(c)
	if err != nil {
		log.Fatal(err)
	}

	znd.ApplyBytesReceived()
	znd.ApplyNeighborAdded()
	znd.ApplyNeighborRemoved()

	znd.Start(true)

	return znd
}
