//send text to cloudgui
package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net/rpc"
	"net/rpc/jsonrpc"
	"os"

	"github.com/UniversityofTromso/mrclean"
)

var (
	//Core component RPC server address
	corerpc string
	//netconn is the transport protocol for the connection
	netconn string

	client *rpc.Client
	err    error
)

func init() {
	flag.StringVar(&corerpc,
		"corerpc", mrclean.CoreAddr,
		"IP:PORT of the Core RPC server")
	flag.StringVar(&netconn, "net", "tcp", "Specifies the connection protocol: tcp, udp, unix etc..")
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func main() {
	flag.Parse()
	client, err = jsonrpc.Dial(netconn, corerpc)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	//fmt.Println("Exanmple of message: ", string(buff))

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("cmd>")
	var ret int

	for scanner.Scan() {
		fmt.Println("SEND :", scanner.Text()) // Println will add back the final '\n'
		err := client.Call("Core.Sort", scanner.Text(), &ret)
		if err != nil {
			log.Println(err)
		}
		if ret == -1 {
			log.Printf("something wrong in the sorting")
		}
		fmt.Print("cmd>")
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading standard input:", err)
	}
}
