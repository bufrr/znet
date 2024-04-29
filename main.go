package main

import (
	"github.com/nknorg/nnet/node"
	"golang.org/x/crypto/sha3"
	"log"
	"time"
	"znet/dht"
	"znet/znode"
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

		c := znode.Config{
			Transport: "tcp",
			P2pPort:   uint16(p2p),
			Keypair:   keypair,
			WsPort:    uint16(ws),
		}

		znd, err := znode.NewZnode(c)
		if err != nil {
			log.Fatal(err)
		}

		znd.Nnet.MustApplyMiddleware(node.BytesReceived{Func: func(msg, msgID, srcID []byte, remoteNode *node.RemoteNode) ([]byte, bool) {
			log.Printf("Receive message \"%s\" from %x by %x", string(msg), srcID, remoteNode.Id)

			_, err = znd.Nnet.SendBytesRelayReply(msgID, []byte("Well received!"), srcID)
			if err != nil {
				log.Fatal(err)
			}

			return msg, true
		}})

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

	time.Sleep(3 * time.Second)

	resp, id, err := znets[0].Nnet.SendBytesRelaySync([]byte("Hello"), znets[len(znets)-1].Nnet.GetLocalNode().Id)
	if err != nil {
		return
	}
	log.Printf("Response: %s from %x", string(resp), id)

	select {}
}
