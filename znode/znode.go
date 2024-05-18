package znode

import (
	"encoding/hex"
	"errors"
	"github.com/bufrr/net"
	"github.com/bufrr/net/node"
	"github.com/bufrr/net/overlay/chord"
	"github.com/bufrr/znet/config"
	"github.com/bufrr/znet/dht"
	pb "github.com/bufrr/znet/protos"
	"google.golang.org/protobuf/proto"
	"log"
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

func (z *Znode) handleZMsg(msg []byte) {
	zmsg := new(pb.ZMessage)
	err := proto.Unmarshal(msg, zmsg)
	if err != nil {
		log.Fatal(err)
	}

	_, _, err = z.Nnet.SendBytesRelaySync(msg, zmsg.To)
	if err != nil {
		log.Fatal(err)
	}
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
		//innerMsg := new(pb.Innermsg)
		//err := proto.Unmarshal(msg, innerMsg)
		//if err != nil {
		//	log.Fatal(err)
		//}
		//
		//switch innerMsg.Identity {
		//case pb.Identity_IDENTITY_SERVER:
		//	_, err = z.Nnet.SendBytesBroadcastAsync(msg, protobuf.BROADCAST_TREE)
		//	if err != nil {
		//		log.Fatal(err)
		//	}
		//case pb.Identity_IDENTITY_CLIENT:
		//	fmt.Printf("###Msg received: %x\n", z.Nnet.GetLocalNode().Id)
		//	to := hex.EncodeToString(innerMsg.Message.To)
		//	if ch, ok := z.msgBuffer[to]; ok {
		//		ch <- innerMsg.Message.Data
		//	} else {
		//		ch := make(chan []byte, 100)
		//		ch <- innerMsg.Message.Data
		//		z.msgBuffer[to] = ch
		//	}
		//
		//	resp, err := z.ReqVlc(msg)
		//	if err != nil {
		//		log.Fatal(err)
		//	}
		//	err = proto.Unmarshal(resp, innerMsg)
		//
		//	innerMsg.Identity = pb.Identity_IDENTITY_SERVER
		//	msg, err = proto.Marshal(innerMsg)
		//	if err != nil {
		//		log.Fatal(err)
		//	}
		//
		//	_, err = z.Nnet.SendBytesRelayReply(msgID, resp, srcID)
		//	if err != nil {
		//		log.Fatal(err)
		//	}
		//}

		zmsg := new(pb.ZMessage)
		err := proto.Unmarshal(msg, zmsg)
		if err != nil {
			log.Fatal(err)
		}

		//_, err = z.Nnet.SendBytesRelayReply(msgID, []byte{}, srcID)
		//if err != nil {
		//	log.Fatal(err)
		//}

		id := hex.EncodeToString(zmsg.To)
		if _, ok := z.msgBuffer[id]; !ok {
			z.msgBuffer[id] = make(chan []byte, 100)
		}
		z.msgBuffer[id] <- zmsg.Data

		log.Printf("Receive message from %x by %x", srcID, remoteNode.Id)
		log.Printf("ws port: %d", z.config.WsPort)
		log.Printf("data: %x", zmsg.Data)

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
