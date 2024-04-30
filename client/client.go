package main

import (
	"bytes"
	"encoding/binary"
	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
	"log"
	pb "znet/protos"
)

func main() {
	// A connection is made to the server
	c, _, err := websocket.DefaultDialer.Dial("ws://localhost:8080", nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	// We prepare our ZMessage
	zMsg := &pb.ZMessage{
		Action:   pb.ZAction_Z_TYPE_READ,
		Data:     []byte("Hello, Server!"),
		Identity: pb.ZIdentity_U_TYPE_CLI,
	}

	// The ZMessage has to be serialized to bytes to be sent over the network
	data, err := proto.Marshal(zMsg)
	if err != nil {
		log.Fatal("marshaling error: ", err)
	}
	// For network transmission, we convert the data to BigEndian
	size := int64(len(data))
	buf := bytes.NewBuffer(make([]byte, 0, size+4))
	binary.Write(buf, binary.BigEndian, size)
	buf.Write(data)

	// Sending our ZMessage to the server
	err = c.WriteMessage(websocket.BinaryMessage, buf.Bytes())
	if err != nil {
		log.Fatal("write:", err)
	}

	// Receipt and printout of the server's response
	_, message, err := c.ReadMessage()
	if err != nil {
		log.Fatal("read:", err)
	}

	log.Printf("Received: %s", message)
}
