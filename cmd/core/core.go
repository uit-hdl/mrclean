package main

import (
	"flag"
	"log"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"os"
	"os/signal"
	"syscall"

	"github.com/folago/mrclean"
)

var (
	rpcserver string
)

func init() {
	flag.StringVar(&rpcserver,
		"rpcserver", ":32123", "IP:PORT of the rpc server, defaults to localhost:32123")
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func main() {
	flag.Parse()
	//TODO use JSON codec
	core := &Core{Visuals: make(map[string]*mrclean.Visual)}
	//rpc.Register(core)
	//l, e := net.Listen("tcp", rpcserver)
	//if e != nil {
	//	log.Fatal("listen error:", e)
	//}
	//go rpc.DefaultServer.Accept(l)
	go serveJSON(core)
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT)
	<-sigc
}

//All method have rpc signature
type Core struct {
	Visuals map[string]*mrclean.Visual
}

//AddVisual adds a visual received fomr the chronicle
func (c *Core) AddVisual(vis *mrclean.Visual, reply *int) error {
	c.Visuals[vis.Name] = vis
	*reply = 0
	log.Println("Added visual ", vis)
	log.Println(c.Visuals)
	return nil
}

//Gsture handle the gestures received from ths users
func (c *Core) Gesture(vis mrclean.Gesture, reply *int) error {
	//TODO sort stuff
	return nil
}

func serveJSON(core *Core) {
	rpc.Register(core)
	l, e := net.Listen("tcp", rpcserver)
	if e != nil {
		log.Fatal("listen error:", e)
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatal(err)
		}

		go rpc.DefaultServer.ServeRequest(jsonrpc.NewServerCodec(conn))
	}
}
