package main

import (
	"flag"
	"fmt"
	"github.com/nknorg/nnet/node"
	"github.com/nknorg/nnet/protobuf"
	"google.golang.org/protobuf/proto"
	"log"
	"znet/config"
	"znet/dht"
	pb "znet/protos"
	"znet/znode"
)

func main() {
	p2pPort := flag.Uint("p2p", config.DEFAULT_P2P_PORT, "p2p port")
	wsPort := flag.Uint("ws", config.DEFAULT_WS_PORT, "websocket port")
	vlcAddr := flag.String("vlc", config.DEFAULT_VLC_ADDR, "vlc address")
	id := flag.String("id", "", "node id")
	remote := flag.String("remote", "", "remote node address")
	flag.Parse()

	keypair, err := dht.GenerateKeyPair([]byte(*id))
	if err != nil {
		log.Fatal(err)
	}

	conf := config.Config{
		Transport: "tcp",
		P2pPort:   uint16(*p2pPort),
		Keypair:   keypair,
		WsPort:    uint16(*wsPort),
		UdpPort:   8080,
		VlcAddr:   *vlcAddr,
	}

	znd, err := znode.NewZnode(conf)
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

	isCreate := len(*remote) == 0
	err = znd.Start(isCreate)
	if err != nil {
		log.Fatal(err)
	}
	if !isCreate {
		err = znd.Nnet.Join(*remote)
		if err != nil {
			log.Fatal(err)
		}
	}

	fmt.Println("addr:", znd.Nnet.GetLocalNode().Addr)

	select {}
}
