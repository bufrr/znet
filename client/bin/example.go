package main

import (
	"encoding/hex"
	"fmt"
	"github.com/bufrr/net/util"
	"github.com/bufrr/znet/client"
	pb "github.com/bufrr/znet/protos"
	"google.golang.org/protobuf/proto"
	"log"
	"os"
)

func main() {
	client1 := client.NewClient([]byte("test1"))
	client2 := client.NewClient([]byte("test5"))
	err := client1.Connect()
	if err != nil {
		log.Fatal(err)
	}
	err = client2.Connect()
	if err != nil {
		log.Fatal(err)
	}

	//addr1 := client1.Address()
	addr2 := client2.Address()

	data := randomMsg(addr2)

	err = client1.Send(addr2, data)
	if err != nil {
		log.Fatal(err)
	}

	msg := <-client2.Receive

	chat := new(pb.ZChat)
	err = proto.Unmarshal(msg, chat)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("msg: ", chat.MessageData)
}

func randomMsg(to string) []byte {
	toBytes, _ := hex.DecodeString(to)
	id, _ := util.RandBytes(32)
	id2, _ := util.RandBytes(32)
	v := make(map[string]uint64)
	v[hex.EncodeToString(id)] = 1
	cc := pb.Clock{Values: v}

	ci := pb.ClockInfo{
		Clock:     &cc,
		NodeId:    id,
		MessageId: id2,
		Count:     2,
		CreateAt:  1243,
	}

	r := []byte("hello hetu! " + string(rune(os.Getpid())))
	chat := pb.ZChat{
		MessageData: string(r),
		Clock:       &ci,
	}
	d, err := proto.Marshal(&chat)
	if err != nil {
		log.Fatal(err)
	}

	zMsg := &pb.ZMessage{
		Data: d,
		To:   toBytes,
		Type: pb.ZType_Z_TYPE_ZCHAT,
	}

	data, err := proto.Marshal(zMsg)
	if err != nil {
		log.Fatal(err)
	}

	return data
}
