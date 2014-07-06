// Chronicle is a aomponent that monitors a directory in the file sysems
//and provide a chronicle of the changes in it.
package main

import (
	"flag"
	"log"
	"net/rpc/jsonrpc"

	"github.com/folago/mrclean"
)

// Chronicle is the type recording the changes in the file system. Its method
// have the signature to be exported via RPC.
type Chronicle struct {
}

var (
	rpcserver string
	session   string
)

func init() {
	flag.StringVar(&session,
		"session", "mrclean", "Name of the session.")
	flag.StringVar(&rpcserver,
		"rpcserver", ":32124", "IP:PORT of the rpc server, defaults to localhost:32124")
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func main() {
	flag.Parse()
	client, err := jsonrpc.Dial("tcp", ":32123")
	if err != nil {
		log.Fatal("dialing:", err)
	}
	var reply int
	args := mrclean.Visual{Name: "scatterplot.jpg"}
	err = client.Call("Core.AddVisual", args, &reply)
	if err != nil {
		log.Fatal("arith error:", err)
	}
	log.Println("Got reply from Core: ", reply)
}
