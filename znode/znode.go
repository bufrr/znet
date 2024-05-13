package znode

import (
	"fmt"
	"github.com/bufrr/net"
	"github.com/bufrr/net/node"
	"github.com/bufrr/net/protobuf"
	"github.com/bufrr/znet/config"
	"github.com/bufrr/znet/dht"
	pb "github.com/bufrr/znet/protos"
	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"
)

type Znode struct {
	Nnet    *nnet.NNet
	keyPair dht.KeyPair
	vlcConn *net.UDPConn
	buf     [65536]byte
	config  config.Config
	wsToVlc chan *pb.Innermsg
	VlcTows chan *pb.Innermsg
}

func NewZnode(c config.Config) (*Znode, error) {
	nn, err := Create(c.Transport, c.P2pPort, c.Keypair.Id())
	if err != nil {
		return nil, err
	}
	udpAddr, err := net.ResolveUDPAddr("udp", c.VlcAddr)
	if err != nil {
		log.Fatal("ResolveUDPAddr err:", err)
	}

	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		log.Fatal("DialUDP err:", err)
	}
	b := make([]byte, 65536)
	wsToVlc := make(chan *pb.Innermsg, 100)
	vlcTows := make(chan *pb.Innermsg, 100)

	return &Znode{
		Nnet:    nn,
		keyPair: c.Keypair,
		vlcConn: conn,
		buf:     [65536]byte(b),
		config:  c,
		wsToVlc: wsToVlc,
		VlcTows: vlcTows,
	}, nil
}

func (z *Znode) Start(isCreate bool) error {
	go z.startWs()
	return z.Nnet.Start(isCreate)
}

func Create(transport string, port uint16, id []byte) (*nnet.NNet, error) {
	conf := &nnet.Config{
		Port:                  port,
		Transport:             transport,
		BaseStabilizeInterval: 500 * time.Millisecond,
	}

	nn, err := nnet.NewNNet(id, conf)
	if err != nil {
		return nil, err
	}

	return nn, nil
}

func (z *Znode) ReqVlc(b []byte) ([]byte, error) {
	_, err := z.vlcConn.Write(b)
	if err != nil {
		return nil, err
	}
	n, _, err := z.vlcConn.ReadFromUDP(z.buf[:])
	if err != nil {
		return nil, err
	}

	return z.buf[:n], nil
}

func (z *Znode) handleZMsg(msg []byte) {
	innerMsg := new(pb.Innermsg)
	err := proto.Unmarshal(msg, innerMsg)
	if err != nil {
		log.Fatal(err)
	}

	_, _, err = z.Nnet.SendBytesRelaySync(msg, innerMsg.Message.To)
	if err != nil {
		log.Fatal(err)
	}
}

func (z *Znode) vlc(w http.ResponseWriter, r *http.Request) {
	var upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()

	go func() {
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("ws read err:", err)
				return
			}

			z.handleZMsg(message)
		}
	}()

	go func() {
		for {
			select {
			case msg := <-z.VlcTows:
				err = c.WriteMessage(websocket.BinaryMessage, msg.Message.Data)
				if err != nil {
					log.Println("ws write err:", err)
					return
				}
			}
		}
	}()

	select {}
}

func (z *Znode) startWs() {
	port := strconv.Itoa(int(z.config.WsPort))
	http.HandleFunc("/vlc"+port, z.vlc)
	http.ListenAndServe("0.0.0.0:"+port, nil)
}

func ApplyBytesReceived(znd *Znode) {
	znd.Nnet.MustApplyMiddleware(node.BytesReceived{Func: func(msg, msgID, srcID []byte, remoteNode *node.RemoteNode) ([]byte, bool) {
		innerMsg := new(pb.Innermsg)
		err := proto.Unmarshal(msg, innerMsg)
		if err != nil {
			log.Fatal(err)
		}

		switch innerMsg.Identity {
		case pb.Identity_IDENTITY_SERVER:
			_, err = znd.Nnet.SendBytesBroadcastAsync(msg, protobuf.BROADCAST_TREE)
			if err != nil {
				log.Fatal(err)
			}
		case pb.Identity_IDENTITY_CLIENT:
			znd.VlcTows <- innerMsg // send msg to websocket
			fmt.Printf("id: %x\n", znd.Nnet.GetLocalNode().Id)

			resp, err := znd.ReqVlc(msg)
			if err != nil {
				log.Fatal(err)
			}
			err = proto.Unmarshal(resp, innerMsg)

			innerMsg.Identity = pb.Identity_IDENTITY_SERVER
			msg, err = proto.Marshal(innerMsg)
			if err != nil {
				log.Fatal(err)
			}

			_, err = znd.Nnet.SendBytesRelayReply(msgID, resp, srcID)
			if err != nil {
				log.Fatal(err)
			}
		}

		log.Printf("Receive message from %x by %x", srcID, remoteNode.Id)

		return msg, true
	}})
}
