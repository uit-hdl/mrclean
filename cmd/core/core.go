package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"os"
	"os/signal"
	"strings"
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
	//config file name
	configfile string
	//config is map of configuration options
	config map[string]string
)

func init() {
	flag.StringVar(&displayrpc,
		"displayrpc", mrclean.DisplayAddr,
		"IP:PORT of the Display RPC server")
	flag.StringVar(&rpcserver,
		"rpcserver", mrclean.CoreAddr, "IP:PORT of the rpc server, defaults to localhost:32123")
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
	core := &Core{Visuals: make(map[string]*mrclean.Visual)}
	go srunService(core)
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
func (c *Core) Sort(layersorder string, reply *int) error {
	//TODO sort stuff
	layersconf := config["layers"]
	layers := strings.Split(layersconf, "/")
	order := strings.Split(layersorder, "/")
	//chek we are doing things correctly
	if len(layers) != len(order) {
		log.Println("order layer and configured layer mismatch:\n%+v\n%+v\n", order, layers)
		return fmt.Errorf("sorting layers number differs from configuraion")
	}
	// we get the map of the position of each layer in
	// the sorting order
	ordermap := make(map[string]int, len(layers))
	for i, s := range order {
		ordermap[s] = i
	}
	//now check we don't have wrong layers
	for _, s := range layers {
		_, ok := ordermap[s]
		if !ok {
			log.Println("order layer and configured layer mismatch:\n%+v\n%+v\n", order, layers)
			return fmt.Errorf("sorting layers elements differ from configuraion")
		}
	}
	//maps layer position to order position
	swap := make(map[int]int, len(layers))
	for i, l := range layers {
		swap[i] = ordermap[l]
	}

	//get the visuals in an array with the corrct metadata
	visuals := make([]mrclean.Visual, len(c.Visuals), 0)
	for _, v := range c.Visuals {
		//strings for metadata
		metastrings := make([]string, len(layers))
		// layers in the name
		sn := strings.Split(v.Name, string(os.PathSeparator))
		//name layer map
		//nlm := make(map[string]int, len(layers))
		//for i, s := range sn {
		//	nlm[s] = i
		//}
		//assemple metadata swapping position
		//according to the swap map
		for l, o := range swap {
			metastrings[o] = sn[l]
		}
		v.Meta = strings.Join(metastrings, "/")
		visuals = append(visuals, *v)
	}
	//SORT
	By(metaf).Sort(visuals)
	//CALL
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
