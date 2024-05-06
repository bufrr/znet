package config

import "znet/dht"

const DEFAULT_P2P_PORT = 33333
const DEFAULT_WS_PORT = 23333
const DEFAULT_UDP_PORT = 8080
const DEFAULT_VLC_ADDR = "127.0.0.1:8080"

type Config struct {
	Transport string
	P2pPort   uint16
	Keypair   dht.KeyPair
	WsPort    uint16
	UdpPort   uint16
	VlcAddr   string
}
