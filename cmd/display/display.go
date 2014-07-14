package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"os"
	"os/signal"
	"syscall"

	geom "github.com/folago/googlmath"
	"github.com/folago/mrclean"
	"github.com/folago/mrclean/displaycloud"
)

var (
	rpcserver       string
	displaycloudurl string
)

func init() {
	flag.StringVar(&displaycloudurl,
		"displaycloudurl", "ws://10.1.255.77:8088/ws_rpc_events",
		"URL of the websocket for displaycloud, default ws://10.1.255.77:8088/ws_rpc_events")
	flag.StringVar(&rpcserver,
		"rpcserver", mrclean.DisplayAddr,
		"IP:PORT of the rpc server, defaults to localhost:32123")
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func main() {
	flag.Parse()
	display := &Display{
		Name: "rocksvv",
		//		Visuals: make(map[string]*mrclean.Visual),
	}
	cli, err := displaycloud.Dial(displaycloudurl)
	if err != nil {
		log.Fatal(err)
	}
	display.client = cli
	//rpc.Register(core)
	//l, e := net.Listen("tcp", rpcserver)
	//if e != nil {
	//	log.Fatal("listen error:", e)
	//}
	//go rpc.DefaultServer.Accept(l)
	go runService(display)
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT)
	<-sigc
}

// Display is the service exposed via rpc
type Display struct {
	Name      string
	Rectangle geom.Rectangle
	client    *displaycloud.Client
	//	Visuals map[string]*mrclean.Visual
}

// The Display methods signatures follow the RPC rules
// See http://golang.org/pkg/net/rpc/
func (d *Display) AddVisual(vis mrclean.Visual, reply *mrclean.Visual) error {
	dcvis, err := d.client.AddVisual(vis)
	if err != nil {
		return err
	}
	//fill the remaining fields
	vis.Origin = dcvis.Origin
	vis.Size = dcvis.Size
	reply = &vis
	return nil //fmt.Errorf("not implemented")
}

//set the origin of the visuald according to the slice of VisualOrigins passed
func (d *Display) SetVisualsOrigin(viso mrclean.VisualOrigins, reply *int) error {
	err := d.client.SetVisualsOrigin(viso.Vids, viso.Origins)
	if err != nil {
		return err
	}
	*reply = 0
	return nil //fmt.Errorf("not implemented")
}

func (d *Display) Size(flag int, reply *[2]float64) error {
	reply = &[2]float64{
		float64(d.Rectangle.Width),
		float64(d.Rectangle.Height),
	}
	return nil
}

func runService(display *Display) {
	rpc.Register(display)
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

func (d *Display) RemoveVisualByID(vid int, reply *int) error {
	return fmt.Errorf("not implemented")
}

func (d *Display) RemoveVisualByName(vname string, reply *int) error {
	return fmt.Errorf("not implemented")
}

func (d *Display) MoveVisual(vid int, reply *int) error {
	return fmt.Errorf("not implemented")
}
