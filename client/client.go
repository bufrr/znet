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
	to, _ := hex.DecodeString("3724b4e85737f7a77b18737535cecd676db38e88514bf0387c2d8fa62905f8eb") // ws://127.0.0.1:23335/vlc23335

	for {
		id, _ := util.RandBytes(32)
		id2, _ := util.RandBytes(32)
		v := make(map[string]uint64)
		v[hex.EncodeToString(id)] = 1
		cc := pb.Clock{Values: v}

		ci := pb.ClockInfo{
			Clock:     &cc,
			Id:        id,
			MessageId: id2,
			Count:     2,
			CreateAt:  1243,
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
			Data: d,
			To:   to,
			Type: pb.ZType_Z_TYPE_ZCHAT,
		}

		innerMsg := &pb.Innermsg{
			Identity: pb.Identity_IDENTITY_CLIENT,
			Action:   pb.Action_ACTION_WRITE,
			Message:  zMsg,
		}

		// The ZMessage has to be serialized to bytes to be sent over the network
		data, err := proto.Marshal(innerMsg)
		if err != nil {
			log.Fatal("marshaling error: ", err)
		}

		// Sending our ZMessage to the server
		err = c.WriteMessage(websocket.BinaryMessage, data)
		if err != nil {
			log.Fatal("write err:", err)
		}

		//time.Sleep(1 * time.Second)
		select {}
	}
}
