package main

import (
	"fmt"
	"github.com/bufrr/znet/config"
	"github.com/bufrr/znet/dht"
	"github.com/bufrr/znet/znode"
	"golang.org/x/crypto/sha3"
	"log"
	"time"
)

func main() {
	p2pPort := 33333
	wsPort := 23333

	znets := make([]*znode.Znode, 0)

	for i := 0; i < 10; i++ {
		h := sha3.New256().Sum([]byte("Hello" + string(rune(i))))
		keypair, _ := dht.GenerateKeyPair(h[:32])

		p2p := p2pPort + i
		ws := wsPort + i

		c := config.Config{
			Transport: "tcp",
			P2pPort:   uint16(p2p),
			Keypair:   keypair,
			WsPort:    uint16(ws),
			UdpPort:   config.DefaultUdpPort,
			VlcAddr:   "127.0.0.1:8050",
		}

		znd, err := znode.NewZnode(c)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("port: %d, wsport: %d, id: %x\n", p2p, ws, znd.Nnet.GetLocalNode().Id)

		znode.ApplyBytesReceived(znd)

		znets = append(znets, znd)
	}

	for i := 0; i < len(znets); i++ {
		time.Sleep(112358 * time.Microsecond)

		err := znets[i].Start(i == 0)
		if err != nil {
			log.Fatal(err)
			return
		}

		if i > 0 {
			err = znets[i].Nnet.Join(znets[0].Nnet.GetLocalNode().Addr)
			if err != nil {
				log.Fatal(err)
				return
			}
		}
	}

	select {}
}
