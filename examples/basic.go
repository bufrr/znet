package main

import (
	"fmt"
	"github.com/bufrr/net/node"
	"github.com/bufrr/net/protobuf"
	"github.com/bufrr/znet/config"
	"github.com/bufrr/znet/dht"
	pb "github.com/bufrr/znet/protos"
	"github.com/bufrr/znet/znode"
	"golang.org/x/crypto/sha3"
	"google.golang.org/protobuf/proto"
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
			UdpPort:   8080,
			VlcAddr:   "127.0.0.1:8080",
		}

		znd, err := znode.NewZnode(c)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("port: %d, wsport: %d, id: %x\n", p2p, ws, znd.Nnet.GetLocalNode().Id)

		znd.Nnet.MustApplyMiddleware(node.BytesReceived{Func: func(msg, msgID, srcID []byte, remoteNode *node.RemoteNode) ([]byte, bool) {
			zmsg := new(pb.ZMessage)
			err := proto.Unmarshal(msg, zmsg)
			if err != nil {
				log.Fatal(err)
			}

			switch zmsg.Identity {
			case pb.ZIdentity_U_TYPE_SER:
				_, err = znd.Nnet.SendBytesBroadcastAsync(msg, protobuf.BROADCAST_TREE)
				if err != nil {
					log.Fatal(err)
				}
			case pb.ZIdentity_U_TYPE_CLI:
				znd.VlcTows <- zmsg // send msg to websocket

				resp, err := znd.ReqVlc(msg)
				if err != nil {
					log.Fatal(err)
				}
				err = proto.Unmarshal(resp, zmsg)

				zmsg.Identity = pb.ZIdentity_U_TYPE_SER
				msg, err = proto.Marshal(zmsg)
				if err != nil {
					log.Fatal(err)
				}

				_, err = znd.Nnet.SendBytesRelayReply(msgID, resp, srcID)
				if err != nil {
					log.Fatal(err)
				}
			}

			log.Printf("Receive message \"%s\" from %x by %x", string(zmsg.Data), srcID, remoteNode.Id)

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

	select {}
}
