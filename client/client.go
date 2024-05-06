package main

import (
	"encoding/hex"
	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
	"log"
	"time"
	pb "znet/protos"
)

func main() {
	c, _, err := websocket.DefaultDialer.Dial("ws://localhost:23334/vlc23334", nil) // id: 406b4c9bb2117df0505a58c6c44a99c8817b7639d9c877bdbea5a8e4e0412740
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()
	to, _ := hex.DecodeString("21665eac30d557de49bc9e22867a8c00eaeb515ebc4cb66aae27873d1aeb196b") // ws://127.0.0.1:23335/vlc23335

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
