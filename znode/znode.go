package znode

import (
	"encoding/hex"
	"errors"
	"github.com/bufrr/net"
	"github.com/bufrr/net/node"
	"github.com/bufrr/net/overlay/chord"
	"github.com/bufrr/net/overlay/routing"
	protobuf "github.com/bufrr/net/protobuf"
	"github.com/bufrr/net/util"
	"github.com/bufrr/znet/config"
	"github.com/bufrr/znet/dht"
	pb "github.com/bufrr/znet/protos"
	"google.golang.org/protobuf/proto"
	"log"
	"math/rand"
	"net"
	"strconv"
	"sync"
	"time"
)

type NodeData struct {
	sync.RWMutex `json:"-"`
	StarTime     time.Time `json:"-"`

	RpcDomain string `json:"rpcDomain"`
	WsDomain  string `json:"wsDomain"`
	RpcPort   uint16 `json:"rpcPort"`
	WsPort    uint16 `json:"wsPort"`
	PublicKey []byte `json:"publicKey"`
}

func NewNodeData(domain string, rpcPort uint16, wsPort uint16) *NodeData {
	return &NodeData{
		RpcDomain: domain,
		WsDomain:  domain,
		RpcPort:   rpcPort,
		WsPort:    wsPort,
		StarTime:  time.Now(),
	}
}

type Znode struct {
	Neighbors map[string]*NodeData
	Nnet      *nnet.NNet
	keyPair   dht.KeyPair
	vlcConn   *net.UDPConn
	buf       [65536]byte
	config    config.Config
	msgBuffer map[string]chan []byte
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

	nd := &pb.NodeData{
		WebsocketPort: uint32(c.WsPort),
		JsonRpcPort:   uint32(c.RpcPort),
		Domain:        c.Domain,
	}

	nn.GetLocalNode().Node.Data, err = proto.Marshal(nd)
	if err != nil {
		log.Fatal(err)
	}

	neighbors := make(map[string]*NodeData)
	return &Znode{
		Nnet:      nn,
		keyPair:   c.Keypair,
		vlcConn:   conn,
		buf:       [65536]byte(b),
		config:    c,
		Neighbors: neighbors,
		msgBuffer: make(map[string]chan []byte),
	}, nil
}

func (z *Znode) GetConfig() *config.Config {
	return &z.config
}

func (z *Znode) Start(isCreate bool) error {
	wsServer := NewWsServer(z)
	go wsServer.Start()
	rpcServer := NewRpcServer(z)
	go rpcServer.Start()
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

func (z *Znode) Id() string {
	return hex.EncodeToString(z.keyPair.Id())
}

func (z *Znode) handleWsZMsg(msg []byte) error {
	msgId, _ := util.RandBytes(32)
	c, _ := z.getClock()
	pbc := pb.Clock{Values: c}

	ci := pb.ClockInfo{
		Clock:     &pbc,
		NodeId:    z.keyPair.Id(),
		MessageId: msgId,
		Count:     2,
		CreateAt:  1243,
	}

	out := new(pb.OutboundMsg)
	err := proto.Unmarshal(msg, out)
	if err != nil {
		return err
	}

	chat := pb.ZChat{
		MessageData: hex.EncodeToString(out.Data),
		Clock:       &ci,
	}
	d, err := proto.Marshal(&chat)
	if err != nil {
		return err
	}

	zMsg := &pb.ZMessage{
		Id:   out.Id,
		Data: d,
		To:   out.To,
		Type: pb.ZType_Z_TYPE_ZCHAT,
	}

	b, _ := proto.Marshal(zMsg)

	_, _, err = z.Nnet.SendBytesRelaySync(b, zMsg.To)
	if err != nil {
		return err
	}
	return nil
}

func (z *Znode) getClock() (map[string]uint64, error) {
	v := rand.Int63()
	return map[string]uint64{
		z.Id(): uint64(v),
	}, nil
}

func (z *Znode) FindWsAddr(key []byte) (string, []byte, error) {
	c, ok := z.Nnet.Network.(*chord.Chord)
	if !ok {
		return "", nil, errors.New("overlay is not chord")
	}
	preds, err := c.FindPredecessors(key, 1)
	if err != nil {
		return "", nil, err
	}
	if len(preds) == 0 {
		return "", nil, errors.New("found no predecessors")
	}

	pred := preds[0]

	nd := new(pb.NodeData)
	err = proto.Unmarshal(pred.Data, nd)

	addr := "ws://" + nd.Domain + ":" + strconv.Itoa(int(nd.WebsocketPort))

	return addr, nd.PublicKey, nil
}

func (z *Znode) ApplyBytesReceived() {
	z.Nnet.MustApplyMiddleware(node.BytesReceived{Func: func(msg, msgID, srcID []byte, remoteNode *node.RemoteNode) ([]byte, bool) {
		zmsg := new(pb.ZMessage)
		err := proto.Unmarshal(msg, zmsg)
		if err != nil {
			log.Fatal(err)
		}

		innerMsg := new(pb.Innermsg)
		innerMsg.Identity = pb.Identity_IDENTITY_CLIENT
		innerMsg.Message = zmsg
		innerMsg.Action = pb.Action_ACTION_WRITE

		msg, err = proto.Marshal(innerMsg)
		if err != nil {
			log.Fatal(err)
		}

		resp, err := z.ReqVlc(msg)
		if err != nil {
			log.Fatal(err)
		}

		innerMsg = new(pb.Innermsg)
		err = proto.Unmarshal(resp, innerMsg)
		if err != nil {
			log.Fatal(err)
		}

		resp, err = proto.Marshal(innerMsg.Message)
		if err != nil {
			log.Fatal(err)
		}

		_, err = z.Nnet.SendBytesRelayReply(msgID, resp, srcID)
		if err != nil {
			log.Printf("SendBytesRelayReply err: %v\n", err)
		}

		id := hex.EncodeToString(zmsg.To)
		if _, ok := z.msgBuffer[id]; !ok {
			z.msgBuffer[id] = make(chan []byte, 100)
		}

		zchat := new(pb.ZChat)
		err = proto.Unmarshal(zmsg.Data, zchat)
		if err != nil {
			log.Printf("parse outbound msg err: %s", err)
			return nil, false
		}

		data, err := hex.DecodeString(zchat.MessageData)
		if err != nil {
			log.Printf("hex decode err: %s", err)
			return nil, false
		}

		z.msgBuffer[id] <- data

		return msg, true
	}})
}

func (z *Znode) ApplyNeighborAdded() {
	z.Nnet.MustApplyMiddleware(chord.NeighborAdded{Func: func(remoteNode *node.RemoteNode, index int) bool {
		nd := new(pb.NodeData)
		err := proto.Unmarshal(remoteNode.Node.Data, nd)
		if err != nil {
			log.Printf("Unmarshal node data: %v\n", err)
			return false
		}
		neighbor := NewNodeData(nd.Domain, uint16(nd.JsonRpcPort), uint16(nd.WebsocketPort))
		z.Neighbors[string(remoteNode.Id)] = neighbor
		return true
	}})
}

func (z *Znode) ApplyNeighborRemoved() {
	z.Nnet.MustApplyMiddleware(chord.NeighborRemoved{Func: func(remoteNode *node.RemoteNode) bool {
		log.Printf("Neighbor %x removed", remoteNode.Id)
		delete(z.Neighbors, string(remoteNode.Id))
		return true
	}})
}

func (z *Znode) ApplyVlcOnRelay() {
	z.Nnet.MustApplyMiddleware(routing.RemoteMessageRouted{
		Func: func(message *node.RemoteMessage, localNode *node.LocalNode, nodes []*node.RemoteNode) (*node.RemoteMessage, *node.LocalNode, []*node.RemoteNode, bool) {
			if message.Msg.MessageType != protobuf.MessageType_BYTES {
				return message, localNode, nodes, false
			}

			b := new(protobuf.Bytes)
			err := proto.Unmarshal(message.Msg.Message, b)
			if err != nil {
				log.Printf("Unmarshal bytes err: %v\n", err)
				return message, localNode, nodes, false
			}

			zMsg := new(pb.ZMessage)
			err = proto.Unmarshal(b.Data, zMsg)
			if err != nil {
				log.Printf("Unmarshal zMsg err: %v\n", err)
				return message, localNode, nodes, false
			}

			if zMsg.Type != pb.ZType_Z_TYPE_ZCHAT {
				log.Printf("zMsg type is not Z_TYPE_ZCHAT")
				return message, localNode, nodes, false
			}

			//zChat := new(pb.ZChat)
			//err = proto.Unmarshal(zMsg.Data, zChat)
			//if err != nil {
			//	log.Printf("Unmarshal zChat err: %v\n", err)
			//	return message, localNode, nodes, false
			//}

			//clockInfo := zChat.Clock

			log.Printf("Receive message %s at node %s from: %s to: %s\n", zMsg.Id, z.Id(), hex.EncodeToString(zMsg.From), hex.EncodeToString(zMsg.To))
			return message, localNode, nodes, true
		},
	})
}
