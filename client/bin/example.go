package main

import (
	"fmt"
	"github.com/bufrr/znet/client"
)

func main() {
	client1 := client.NewClient([]byte("test1"))
	client2 := client.NewClient([]byte("test5"))
	err := client1.Connect()
	if err != nil {
		fmt.Println("err: ", err)
	}
	err = client2.Connect()
	if err != nil {
		fmt.Println("err: ", err)
	}

	go client1.ReadMsg()
	go client2.ReadMsg()

	//addr1 := client1.Address()
	addr2 := client2.Address()

	err = client1.Send(addr2, []byte("Hello, world!"))
	if err != nil {
		fmt.Println("err: ", err)
	}

	fmt.Println("msg: ", string(<-client2.Receive))
}
