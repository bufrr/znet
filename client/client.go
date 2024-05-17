package client

import (
	"encoding/hex"
	"errors"
	"github.com/bufrr/znet/dht"
	pb "github.com/bufrr/znet/protos"
	"github.com/bufrr/znet/utils"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/sha3"
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
	receive chan *pb.ZMessage
	config  *Config
}

func (c *Client) Send(address string, msg []byte) error {
	if c.conn == nil {
		return errors.New("ws not connected")
	}
	return nil
}

func NewClient(seed []byte) *Client {
	h := sha3.New256().Sum(seed)
	keypair, _ := dht.GenerateKeyPair(h[:32])
	recv := make(chan *pb.ZMessage)
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
	dialer := &websocket.Dialer{}
	conn, _, err := dialer.Dial(addr, nil)
	if err != nil {
		return err
	}
	c.conn = conn
	conn.WriteMessage(websocket.TextMessage, c.key.Id())
	return nil
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
