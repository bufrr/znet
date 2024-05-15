package znode

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/bufrr/net"
	"github.com/bufrr/net/node"
	"github.com/bufrr/net/overlay/chord"
	"github.com/bufrr/net/protobuf"
	"github.com/bufrr/znet/config"
	"github.com/bufrr/znet/dht"
	pb "github.com/bufrr/znet/protos"
	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
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
	wsToVlc   chan *pb.Innermsg
	VlcTows   chan *pb.Innermsg
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
		wsToVlc:   wsToVlc,
		VlcTows:   vlcTows,
		Neighbors: neighbors,
	}, nil
}

func (z *Znode) Start(isCreate bool) error {
	go z.startWs()
	go z.startRpc()
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

	return pred.Addr, pred.Id, nil
}

func (z *Znode) ApplyBytesReceived() {
	z.Nnet.MustApplyMiddleware(node.BytesReceived{Func: func(msg, msgID, srcID []byte, remoteNode *node.RemoteNode) ([]byte, bool) {
		innerMsg := new(pb.Innermsg)
		err := proto.Unmarshal(msg, innerMsg)
		if err != nil {
			log.Fatal(err)
		}

		switch innerMsg.Identity {
		case pb.Identity_IDENTITY_SERVER:
			_, err = z.Nnet.SendBytesBroadcastAsync(msg, protobuf.BROADCAST_TREE)
			if err != nil {
				log.Fatal(err)
			}
		case pb.Identity_IDENTITY_CLIENT:
			z.VlcTows <- innerMsg // send msg to websocket
			fmt.Printf("id: %x\n", z.Nnet.GetLocalNode().Id)

			resp, err := z.ReqVlc(msg)
			if err != nil {
				log.Fatal(err)
			}
			err = proto.Unmarshal(resp, innerMsg)

			innerMsg.Identity = pb.Identity_IDENTITY_SERVER
			msg, err = proto.Marshal(innerMsg)
			if err != nil {
				log.Fatal(err)
			}

			_, err = z.Nnet.SendBytesRelayReply(msgID, resp, srcID)
			if err != nil {
				log.Fatal(err)
			}
		}

		log.Printf("Receive message from %x by %x", srcID, remoteNode.Id)

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

func GetExtIp(remote string) (string, error) {
	resp, err := Call(remote, "getExtIp", 0, map[string]interface{}{})
	if err != nil {
		return "", err
	}

	var ret map[string]interface{}
	if err := json.Unmarshal(resp, &ret); err != nil {
		return "", err
	}

	return ret["extIp"].(string), nil
}

func Call(address string, method string, id uint, params map[string]interface{}) ([]byte, error) {
	data, err := json.Marshal(map[string]interface{}{
		"method": method,
		"id":     id,
		"params": params,
	})
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := client.Post(address, "application/json", strings.NewReader(string(data)))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}
