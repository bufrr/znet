package main

import (
	"github.com/gorilla/websocket"
	"github.com/nknorg/nnet/node"
	"github.com/nknorg/nnet/protobuf"
	"golang.org/x/crypto/sha3"
	"google.golang.org/protobuf/proto"
	"log"
	"time"
	"znet/dht"
	pb "znet/protos"
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
			UdpPort:   8080,
			VlcAddr:   "127.0.0.1:8080",
		}

		znd, err := znode.NewZnode(c)
		if err != nil {
			log.Fatal(err)
		}

		znd.Nnet.MustApplyMiddleware(node.BytesReceived{Func: func(msg, msgID, srcID []byte, remoteNode *node.RemoteNode) ([]byte, bool) {
			zmsg := new(pb.ZMessage)
			err := proto.Unmarshal(msg, zmsg)
			if err != nil {
				log.Fatal(err)
			}

			_, err = znd.ReqVlc(msg)
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

	time.Sleep(5 * time.Second)

	c, _, err := websocket.DefaultDialer.Dial("ws://localhost:23333/vlc23333", nil)
	if err != nil {
		log.Fatal("dial err:", err)
	}

	to := znets[2].Nnet.GetLocalNode().Id

	zm := &pb.ZMessage{
		Action:   pb.ZAction_Z_TYPE_READ,
		Data:     []byte("Hello, Server!"),
		Identity: pb.ZIdentity_U_TYPE_CLI,
		To:       to,
	}

	data, err := proto.Marshal(zm)
	if err != nil {
		log.Fatal("marshaling error: ", err)
	}

	err = c.WriteMessage(websocket.BinaryMessage, data)
	if err != nil {
		log.Fatal("write:", err)
	}

	select {}
}
