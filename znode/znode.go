package znode

import (
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/nknorg/nnet"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"
	"znet/dht"
)

type znode struct {
	Nnet    *nnet.NNet
	keyPair dht.KeyPair
	vlcConn *net.UDPConn
	buf     [102400]byte
	config  Config
}

type Config struct {
	Transport string
	P2pPort   uint16
	Keypair   dht.KeyPair
	WsPort    uint16
	udpPort   uint16
	vlcAddr   string
}

func NewZnode(c Config) (*znode, error) {
	nn, err := Create(c.Transport, c.P2pPort, c.Keypair.Id())
	if err != nil {
		return nil, err
	}
	udpAddr, err := net.ResolveUDPAddr("udp", c.vlcAddr)
	if err != nil {
		fmt.Println("Error: ", err)
	}

	conn, err := net.DialUDP("udp", nil, udpAddr)
	b := make([]byte, 102400)

	return &znode{
		Nnet:    nn,
		keyPair: c.Keypair,
		vlcConn: conn,
		buf:     [102400]byte(b),
		config:  c,
	}, nil
}

func (z *znode) Start(isCreate bool) error {
	go z.startWs(isCreate)
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

func (z *znode) readVlc(b []byte) ([]byte, error) {
	_, err := z.vlcConn.Write(b)
	if err != nil {
		return nil, err
	}
	buf := make([]byte, 409600)
	n, _, err := z.vlcConn.ReadFromUDP(z.buf[:])
	if err != nil {
		return nil, err
	}

	return buf[:n], nil
}

func (z *znode) writeVlc(b []byte) error {
	_, err := z.vlcConn.Write(b)
	return err
}

func (z *znode) vlc(w http.ResponseWriter, r *http.Request) {
	var upgrader = websocket.Upgrader{} // use default options

	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.Println("ws read err:", err)
			break
		}

		log.Printf("recv: %s, type: %s", message, mt)

		nbrs, err := z.Nnet.GetLocalNode().GetNeighbors(nil)
		if err != nil {
			log.Println("get neighbors err:", err)
			return
		}

		data, _, err := z.Nnet.SendBytesRelaySync(message, nbrs[0].Id)
		if err != nil {
			log.Println("nnet send err:", err)
			return
		}

		err = c.WriteMessage(mt, data)
		if err != nil {
			log.Println("ws write err:", err)
			break
		}
	}
}

func (z *znode) startWs(isCreate bool) {
	if !isCreate {
		return
	}
	http.HandleFunc("/vlc", z.vlc)
	http.ListenAndServe(":"+strconv.Itoa(int(z.config.WsPort)), nil)
}
