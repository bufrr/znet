package main

import (
	"encoding/hex"
	pb "github.com/bufrr/znet/protos"
	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
	"log"
	"time"
)

func main() {
	c, _, err := websocket.DefaultDialer.Dial("ws://192.168.1.110:23333/vlc23333", nil) // id: 406b4c9bb2117df0505a58c6c44a99c8817b7639d9c877bdbea5a8e4e0412740
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()
	to, _ := hex.DecodeString("d33f8bf08cdc373093850bd4966213cfa19a87c2a2bd4738220fa011c606992f") // ws://127.0.0.1:23335/vlc23335

	for {
		// A connection is made to the server

		// We prepare our ZMessage
		zMsg := &pb.ZMessage{
			Action:   pb.ZAction_Z_TYPE_READ,
			Data:     []byte("Hello, Server!"),
			Identity: pb.ZIdentity_U_TYPE_CLI,
			To:       to,
		}

		// The ZMessage has to be serialized to bytes to be sent over the network
		data, err := proto.Marshal(zMsg)
		if err != nil {
			log.Fatal("marshaling error: ", err)
		}

		// Sending our ZMessage to the server
		err = c.WriteMessage(websocket.BinaryMessage, data)
		if err != nil {
			log.Fatal("write err:", err)
		}

		time.Sleep(1 * time.Second)
	}
}
