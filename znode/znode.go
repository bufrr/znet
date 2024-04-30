package znode

import (
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/nknorg/nnet"
	"google.golang.org/protobuf/proto"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"
	"znet/dht"
	pb "znet/protos"
)

type Znode struct {
	Nnet    *nnet.NNet
	keyPair dht.KeyPair
	vlcConn *net.UDPConn
	buf     [102400]byte
	config  Config
	wsToVlc chan *pb.ZMessage
	VlcTows chan *pb.ZMessage
}

type Config struct {
	Transport string
	P2pPort   uint16
	Keypair   dht.KeyPair
	WsPort    uint16
	UdpPort   uint16
	VlcAddr   string
}

func NewZnode(c Config) (*Znode, error) {
	nn, err := Create(c.Transport, c.P2pPort, c.Keypair.Id())
	if err != nil {
		return nil, err
	}
	udpAddr, err := net.ResolveUDPAddr("udp", c.VlcAddr)
	if err != nil {
		fmt.Println("Error: ", err)
	}

	conn, err := net.DialUDP("udp", nil, udpAddr)
	b := make([]byte, 102400)
	wsToVlc := make(chan *pb.ZMessage, 100)
	vlcTows := make(chan *pb.ZMessage, 100)

	return &Znode{
		Nnet:    nn,
		keyPair: c.Keypair,
		vlcConn: conn,
		buf:     [102400]byte(b),
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
	zMsg := new(pb.ZMessage)
	err := proto.Unmarshal(msg, zMsg)
	if err != nil {
		log.Fatal(err)
	}

	_, _, err = z.Nnet.SendBytesRelaySync(msg, zMsg.To)
	if err != nil {
		log.Fatal(err)
	}
}

func (z *Znode) vlc(w http.ResponseWriter, r *http.Request) {
	var upgrader = websocket.Upgrader{} // use default options

	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()

	go func() {
		for {
			mt, message, err := c.ReadMessage()
			if err != nil {
				log.Println("ws read err:", err)
				break
			}

			z.handleZMsg(message)

			log.Printf("recv: %s, type: %d", message, mt)
		}
	}()

	go func() {
		for {
			select {
			case msg := <-z.VlcTows:
				//data, err := proto.Marshal(msg)
				//if err != nil {
				//	log.Fatal(err)
				//}
				err = c.WriteMessage(websocket.BinaryMessage, msg.Data)
				if err != nil {
					log.Println("ws write err:", err)
					break
				}
			}
		}
	}()

	select {}
}

func (z *Znode) startWs() {
	port := strconv.Itoa(int(z.config.WsPort))
	http.HandleFunc("/vlc"+port, z.vlc)
	http.ListenAndServe(":"+port, nil)
}
