package client

import (
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/bufrr/znet/dht"
	"github.com/bufrr/znet/utils"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/sha3"
	"net"
	"net/url"
)

type Config struct {
	SeedRpcServer []string
}

func NewClientConfig() *Config {
	return &Config{
		SeedRpcServer: []string{"http://127.0.0.1:12345/rpc12345"},
	}
}

type Client struct {
	conn    *websocket.Conn
	key     dht.KeyPair
	Receive chan []byte
	config  *Config
}

func (c *Client) Send(address string, msg []byte) error {
	if c.conn == nil {
		return errors.New("ws not connected")
	}
	//zmsg := new(pb.ZMessage)
	//zmsg.From = c.key.Id()
	//to, err := hex.DecodeString(address)
	//if err != nil {
	//	return err
	//}
	//zmsg.To = to
	//zmsg.Data = msg
	//m, _ := proto.Marshal(zmsg)
	err := c.conn.WriteMessage(websocket.BinaryMessage, msg)
	if err != nil {
		return err
	}
	return nil
}

func NewClient(seed []byte) *Client {
	h := sha3.New256().Sum(seed)
	keypair, _ := dht.GenerateKeyPair(h[:32])
	recv := make(chan []byte)
	c := NewClientConfig()

	return &Client{nil, keypair, recv, c}
}

func (c *Client) Connect() error {
	wsAddr, err := c.GetWsAddr()
	if err != nil {
		return err
	}
	return c.connect(wsAddr)
}

func (c *Client) connect(addr string) error {

	u, _ := url.Parse(addr)

	_, port, _ := net.SplitHostPort(u.Host)

	conn, _, err := websocket.DefaultDialer.Dial(addr+"/ws"+port, nil)
	if err != nil {
		return err
	}
	c.conn = conn
	err = conn.WriteMessage(websocket.TextMessage, c.key.Id())
	if err != nil {
		return err
	}

	go c.readMsg()
	fmt.Println("connected to ", addr)

	return nil
}

func (c *Client) readMsg() {
	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			fmt.Println("read conn err: ", err)
			return
		}
		c.Receive <- msg
	}
}

func (c *Client) Address() string {
	return hex.EncodeToString(c.key.Id())
}

func (c *Client) GetWsAddr() (string, error) {
	wsAddr, err := utils.GetWsAddr(c.config.SeedRpcServer[0], c.Address())
	if err != nil {
		return "", err
	}
	return wsAddr, nil
}
