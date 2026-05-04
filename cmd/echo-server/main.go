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
		return
	}

	for true {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("accept err:", err)
		}
		go func() {
			reader := bufio.NewReader(conn)
			writer := bufio.NewWriter(conn)
			for true {
				line, err := reader.ReadBytes('\n')
				if err != nil {
					fmt.Println("reader err: ", err)
					return
				}
				fmt.Printf("New Echo: %v", string(line))
				_, err = writer.Write(line)
				if err != nil {
					fmt.Println("write err: ", err)
					return
				}
				err = writer.Flush()
				if err != nil {
					fmt.Println("flush err: ", err)
					return
				}
				err = splitlayer.Flush(conn)
				if err != nil {
					fmt.Println("flush conn err: ", err)
					return
				}
			}
		}()
	}
}
