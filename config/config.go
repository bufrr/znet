package config

import "github.com/bufrr/znet/dht"

const DefaultP2pPort = 33333
const DefaultWsPort = 23333
const DefaultUdpPort = 8080
const DefaultVlcAddr = "127.0.0.1:8080"

type Config struct {
	Transport string
	P2pPort   uint16
	Keypair   dht.KeyPair
	WsPort    uint16
	UdpPort   uint16
	VlcAddr   string
}
