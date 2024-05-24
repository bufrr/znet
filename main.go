package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"github.com/bufrr/znet/config"
	"github.com/bufrr/znet/dht"
	"github.com/bufrr/znet/utils"
	"github.com/bufrr/znet/znode"
	"log"
)

func main() {
	p2pPort := flag.Uint("p2p", config.DefaultP2pPort, "p2p port")
	wsPort := flag.Uint("ws", config.DefaultWsPort, "websocket port")
	vlcAddr := flag.String("vlc", config.DefaultVlcAddr, "vlc address")
	rpcPort := flag.Uint("rpc", config.DefaultRpcPort, "rpc address")
	id := flag.String("id", "", "node id")
	remote := flag.String("remote", "", "remote node address")
	domain := flag.String("domain", "", "domain")
	flag.Parse()

	seed := [32]byte{}
	copy(seed[:], *id)
	keypair, err := dht.GenerateKeyPair(seed[:])
	if err != nil {
		log.Fatal(err)
	}

	seedList := []string{*remote}

	conf := config.Config{
		Transport: "tcp",
		P2pPort:   uint16(*p2pPort),
		Keypair:   keypair,
		WsPort:    uint16(*wsPort),
		UdpPort:   8050,
		VlcAddr:   *vlcAddr,
		RpcPort:   uint16(*rpcPort),
		SeedList:  seedList,
		Domain:    *domain,
	}

	if len(*domain) == 0 {
		ip, err := utils.GetExtIp(conf.SeedList[0])
		if err != nil {
			log.Fatal("get ext ip err:", err)
		}
		conf.Domain = ip
	}

	znd, err := znode.NewZnode(conf)
	if err != nil {
		log.Fatal(err)
	}

	zid := hex.EncodeToString(znd.Nnet.GetLocalNode().Id)
	fmt.Println("id:", zid)

	znd.ApplyBytesReceived()
	znd.ApplyNeighborRemoved()
	znd.ApplyNeighborAdded()
	znd.ApplyVlcOnRelay()

	isCreate := len(*remote) == 0
	err = znd.Start(isCreate)
	if err != nil {
		log.Fatal(err)
	}
	if !isCreate {
		err = znd.Nnet.Join(seedList[0])
		if err != nil {
			log.Fatal(err)
		}
	}

	fmt.Println("addr:", znd.Nnet.GetLocalNode().Addr)

	select {}
}
