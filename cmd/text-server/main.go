package main

import (
	"bufio"
	"fmt"
	"time"

	splitlayer "github.com/stay-miku-39/split-socket/pkg/split-layer"
	tcplayer "github.com/stay-miku-39/split-socket/pkg/tcp-layer"
	"github.com/stay-miku-39/split-socket/pkg/utils"
)

func main() {
	utils.SetDefualtLoggerLevel(utils.Info)
	split := splitlayer.NewSplitTransport(&splitlayer.SplitConfig{
		HalfConnectTimeout:     10 * time.Second,
		ReadTimeout:            10 * time.Second,
		WriteTimeout:           10 * time.Second,
		MaxHalfConnectionCount: 128,
		MaxFullConnectionCount: 128,
	})

	split.WithTransport(&tcplayer.TCPTransport{})

	listener, err := split.Listen("localhost:10000,localhost:10001")

	if err != nil {
		fmt.Println("err:", err)
	}

	for true {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("err:", err)
			return
		}
		go func() {
			reader := bufio.NewReader(conn)
			for true {
				line, err := reader.ReadString('\n')
				if err != nil {
					fmt.Println("err: ", err)
					return
				}
				fmt.Println("New line: ", line)
			}
		}()
	}
}
