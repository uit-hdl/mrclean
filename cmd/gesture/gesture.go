package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"

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
	//Core component RPC server address
	corerpc string
	//the core component rpc client
	client *rpc.Client
	//config file name
	configfile string
	//config is map of configuration options
	config map[string]string
)

func init() {
	flag.StringVar(&corerpc,
		"corerpc", mrclean.CoreAddr,
		"IP:PORT of the Core RPC server")
	flag.StringVar(&configfile,
		"configfile", "config.json", "Configuration file for Mr. Clean")
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func main() {
	flag.Parse()
	var err error
	config, err = ReadConfig(configfile)
	if err != nil {
		log.Fatal(err)
	}
	var (
		farg  int
		reply [2]float64
	)
	err = client.Call("Display.Size", farg, &reply)
	if err != nil {
		log.Fatal("Display error:", err)
	}
	core.DispW = reply[0]
	core.DispH = reply[1]

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT)
	<-sigc
}

func ReadConfig(fname string) (map[string]string, error) {
	file, err := os.Open(fname)
	if err != nil {
		return nil, err
	}
	buff, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}
	var config map[string]string
	err = json.Unmarshal(buff, &config)
	if err != nil {
		return nil, err
	}
	log.Printf("Got configuration: %+v\n", config)
	return config, nil
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
