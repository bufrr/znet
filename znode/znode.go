package znode

import (
	"github.com/gorilla/websocket"
	"github.com/nknorg/nnet"
	"log"
	"net/http"
	"strconv"
	"time"
	"znet/dht"
)

type znode struct {
	Nnet    *nnet.NNet
	keyPair dht.KeyPair
	config  Config
}

type Config struct {
	Transport string
	P2pPort   uint16
	Keypair   dht.KeyPair
	WsPort    uint16
	udpPort   uint16
	vlcIp     string
	vlcPort   uint16
}

func NewZnode(c Config) (*znode, error) {
	nn, err := Create(c.Transport, c.P2pPort, c.Keypair.Id())
	if err != nil {
		return nil, err
	}

	return &znode{
		Nnet:    nn,
		keyPair: c.Keypair,
		config:  c,
	}, nil
}

func (z *znode) Start(isCreate bool) error {
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

func (z *znode) readVlc() ([]byte, error) {
	return nil, nil
}

func (z *znode) writeVlc([]byte) error {
	return nil
}

func (z *znode) echo(w http.ResponseWriter, r *http.Request) {
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
			log.Println("read:", err)
			break
		}

		log.Printf("recv: %s, type: %s", message, mt)
		err = c.WriteMessage(mt, message)
		if err != nil {
			log.Println("write:", err)
			break
		}
	}
}

func (z *znode) startWs() {
	http.HandleFunc("/vlc", z.echo)
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(int(z.config.WsPort)), nil))
}
