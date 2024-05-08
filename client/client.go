package main

import (
	"encoding/hex"
	"github.com/bufrr/net/util"
	pb "github.com/bufrr/znet/protos"
	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
	"log"
)

func main() {
	c, _, err := websocket.DefaultDialer.Dial("ws://127.0.0.1:23333/vlc23333", nil) // id: 406b4c9bb2117df0505a58c6c44a99c8817b7639d9c877bdbea5a8e4e0412740
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()
	to, _ := hex.DecodeString("f78e5a39e3d433986c4b8026d0baeb62b7eb845c29bb83a04b79d645ef7efbba") // ws://127.0.0.1:23335/vlc23335

	for {
		cc := pb.Clock{Values: make(map[string]uint64)}
		ci := pb.ClockInfo{
			Clock:     &cc,
			Id:        []byte("1"),
			MessageId: []byte("222"),
			Count:     1,
			CreateAt:  123,
		}

		r, _ := util.RandBytes(8)
		chat := pb.ZChat{
			MessageData: hex.EncodeToString(r),
			Clock:       &ci,
		}
		d, err := proto.Marshal(&chat)
		if err != nil {
			log.Fatal(err)
		}

		// A connection is made to the server
		// We prepare our ZMessage
		zMsg := &pb.ZMessage{
			Action:   pb.ZAction_Z_TYPE_WRITE,
			Data:     d,
			Identity: pb.ZIdentity_U_TYPE_CLI,
			To:       to,
			Type:     pb.ZType_Z_TYPE_ZCHAT,
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

		//time.Sleep(1 * time.Second)
		break
	}
}
