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
	//IP:PORT of thr rpc server
	rpcserver string
	//Display component RPC server address
	displayrpc string
	//the display component rpc client
	client *rpc.Client
)

func init() {
	flag.StringVar(&displayrpc,
		"rpcserver", mrclean.DisplayAddr,
		"IP:PORT of the Display RPC server")
	flag.StringVar(&rpcserver,
		"rpcserver", mrclean.CoreAddr, "IP:PORT of the rpc server, defaults to localhost:32123")
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func main() {
	flag.Parse()
	core := &Core{Visuals: make(map[string]*mrclean.Visual)}
	go srunService(core)
	var err error
	client, err = jsonrpc.Dial("tcp", displayrpc)
	if err != nil {
		log.Fatal("dialing:", err)
	}

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
	log.Printf("Added visual %+v\n", vis)
	log.Println("len(Viausls) ", len(c.Visuals))
	//adding visual to the display
	var rvis *mrclean.Visual
	err := client.Call("Core.AddVisual", *vis, rvis)
	if err != nil {
		return err
	}
	//the rpc result has all the data so we pu that in the map
	c.Visuals[rvis.Name] = rvis
	return nil
}

//Sort handle the gestures received from ths users to sprt the visuals
func (c *Core) Sort(order string, reply *int) error {
	//TODO sort stuff
	return nil
}

//runs the given object as an RPC service using JSON as encoding
func srunService(core *Core) {
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

		go rpc.DefaultServer.ServeCodec(jsonrpc.NewServerCodec(conn))
	}
}
