package main

import (
	"github.com/nknorg/nnet"
	"github.com/nknorg/nnet/node"
	"github.com/nknorg/nnet/util"
	"golang.org/x/crypto/sha3"
	"log"
	"time"
	"znet/dht"
	"znet/znode"
)

func main() {
	h := sha3.New256().Sum([]byte("Hello"))
	keypair, _ := dht.GenerateKeyPair(h[:32])

	p2pPort := 33333
	wsPort := 23333

	c := znode.Config{
		Transport: "tcp",
		P2pPort:   uint16(p2pPort),
		Keypair:   keypair,
		WsPort:    uint16(wsPort),
	}

	znd, err := znode.NewZnode(c)
	if err != nil {
		log.Fatal(err)
	}
	znd.Start(true)

	nnets := make([]*nnet.NNet, 0)

	for i := 1; i <= 10; i++ {
		id, err := util.RandBytes(32)
		if err != nil {
			log.Fatal(err)
			return
		}

		nn, err := znode.Create("tcp", uint16(p2pPort+i), id)
		if err != nil {
			log.Fatal(err)
			return
		}

		nn.MustApplyMiddleware(node.BytesReceived{Func: func(msg, msgID, srcID []byte, remoteNode *node.RemoteNode) ([]byte, bool) {
			log.Printf("Receive message \"%s\" from %x by %x", string(msg), srcID, remoteNode.Id)

			_, err = nn.SendBytesRelayReply(msgID, []byte("Well received!"), srcID)
			if err != nil {
				log.Fatal(err)
			}

			return msg, true
		}})

		//nn.MustApplyMiddleware(routing.RemoteMessageReceived{Func: func(msg *node.RemoteMessage) (*node.RemoteMessage, bool) {
		//	log.Printf("Receive message \"%s\" from %x by %x", string(msg.Msg.Message), msg.RemoteNode.Id)
		//
		//	msgId, _ := util.RandBytes(32)
		//	_, _, err = nn.SendMessageSync(&protobuf.Message{
		//		RoutingType: protobuf.DIRECT,
		//		MessageType: protobuf.BYTES,
		//		Message:     []byte("Well received"),
		//		MessageId:   msgId,
		//		ReplyToId:   msg.Msg.MessageId,
		//		SrcId:       msg.Msg.DestId,
		//		DestId:      msg.Msg.SrcId,
		//	}, protobuf.RELAY, time.Second)
		//	if err != nil {
		//		log.Fatal(err)
		//	}
		//
		//	return msg, true
		//}})

		nnets = append(nnets, nn)
	}

	for i := 0; i < len(nnets); i++ {
		time.Sleep(112358 * time.Microsecond)

		err = nnets[i].Start(i == 0)
		if err != nil {
			log.Fatal(err)
			return
		}

		err = nnets[i].Join(znd.Nnet.GetLocalNode().Addr)
		if err != nil {
			log.Fatal(err)
			return
		}
	}

	resp, id, err := znd.Nnet.SendBytesRelaySync([]byte("Hello"), nnets[0].GetLocalNode().Id)
	if err != nil {
		return
	}
	log.Printf("Response: %s from %x", string(resp), id)

	//znd.Nnet.SendMessageSync(&protobuf.Message{
	//	RoutingType: protobuf.DIRECT,
	//	MessageType: protobuf.BYTES,
	//	Message:     []byte("Well received"),
	//	MessageId:   []byte("123"),
	//	ReplyToId:   []byte("456"),
	//	SrcId:       znd.Nnet.GetLocalNode().Id,
	//	DestId:      nnets[0].GetLocalNode().Id,
	//}, protobuf.RELAY, time.Second)

	select {}
}
