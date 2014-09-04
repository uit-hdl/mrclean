package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/UniversityofTromso/mrclean"
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
	//netconn is the transport protocol for the connection
	netconn string
)

func init() {
	flag.StringVar(&displayrpc,
		"displayrpc", mrclean.DisplayAddr,
		"IP:PORT of the Display RPC server")
	flag.StringVar(&rpcserver,
		"rpcserver", mrclean.CoreAddr, "IP:PORT of the rpc server, defaults to localhost:32123")
	flag.StringVar(&configfile,
		"configfile", "config.json", "Configuration file for Mr. Clean")
	flag.StringVar(&netconn, "net", "tcp", "Specifies the connection protocol: tcp, udp, unix etc..")
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
	client, err = jsonrpc.Dial(netconn, displayrpc)
	if err != nil {
		log.Fatal("dialing:", err)
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
	log.Printf("core.DispW: %f, core.DispH: %f\n", core.DispW, core.DispH)

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT)
	<-sigc
}

//All method have rpc signature
type Core struct {
	Visuals      map[string]*mrclean.Visual
	DispW, DispH float64
	Lock         sync.Mutex
}

//AddVisual adds a visual received from the chronicle
func (c *Core) AddVisual(vis *mrclean.Visual, reply *int) error {
	c.Lock.Lock()
	defer c.Lock.Unlock()
	//c.Visuals[vis.Name] = vis
	*reply = 0
	log.Printf("Received visual %+v\n", vis)
	//adding visual to the display
	rvis := mrclean.Visual{
	//Origin: make([]float64, 2),
	//Size:   make([]float64, 2),
	}
	err := client.Call("Display.AddVisual", vis, &rvis)
	if err != nil {
		log.Println(err)
		//maye just return nil? chronicle cannot do much a this point
		*reply = -1
		return err
	}
	//the rpc result has the missing data so we update the visual and put it in the map
	vis.Origin = rvis.Origin
	vis.Size = rvis.Size
	vis.ID = rvis.ID
	c.Visuals[vis.Name] = vis
	log.Printf("Added visual %+v\n", vis)
	log.Println("len(Visuals) ", len(c.Visuals))
	return nil
}

//Sort handle the gestures received from ths users to sprt the visuals
func (c *Core) Sort(layersorder string, reply *int) error {
	c.Lock.Lock()
	defer c.Lock.Unlock()
	layersconf := config["layers"]
	layers := strings.Split(layersconf, "/")
	order := strings.Split(layersorder, "/")
	//chek we are doing things correctly
	if len(layers) != len(order) {
		log.Println("order layer and configured layer mismatch:\n%+v\n%+v\n", order, layers)
		*reply = -1
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
			log.Printf("order layer and configured layer mismatch:\n%+v\n%+v\n", order, layers)
			*reply = -1
			return fmt.Errorf("sorting layers elements differ from configuraion")
		}
	}
	//maps layer position to order position
	swap := make(map[int]int, len(layers))
	for i, l := range layers {
		swap[i] = ordermap[l]
	}
	var dx, dy float64

	//get the visuals in an array with the correct meta-data
	visuals := make([]mrclean.Visual, 0, len(c.Visuals))
	for _, v := range c.Visuals {
		//strings for metadata
		metastrings := make([]string, len(layers))
		// layers in the name
		log.Printf("splitting for metadata: %s", v.Name)
		sn := strings.Split(v.Name, string(os.PathSeparator))
		if len(sn) != len(metastrings) {
			log.Printf("WARNING: metadata and path of images are of different length, path is %d and shuld be %d\n",
				len(sn), len(metastrings))
		}
		//name layer map
		//nlm := make(map[string]int, len(layers))
		//for i, s := range sn {
		//	nlm[s] = i
		//}
		log.Printf(" len(sn) = %d len(metastrings) = %d", len(sn), len(metastrings))
		//assemble meta-data swapping position
		//according to the swap map
		for l, o := range swap {
			log.Printf("swapping metastrings[%v] = sn[%v]\n", o, l)
			metastrings[o] = sn[l]
		}
		v.Meta = strings.Join(metastrings, "/")
		visuals = append(visuals, *v)
		//get the bigger visual to use as placeholder
		//for the displaying
		dx = math.Max(dx, v.Size[0])
		dy = math.Max(dy, v.Size[1])
	}
	log.Printf("sorting: len(visuals) = %v\n", len(visuals))
	//SORT
	By(metaf).Sort(visuals)
	//loop to put the visuals on screen
	// evenly spaced and sorted
	dx += 0.05 //5 cm
	dy += 0.05 //5 cm
	var (
		lastpx                         = -c.DispW*0.5 + dx*0.5
		lastpy                         = c.DispH*0.5 - dy*0.5
		origins *mrclean.VisualOrigins = mrclean.NewVisualOrigins()
		row     float64
	)
	log.Printf("dx: %f, dy: %f\n", dx, dy)
	for i := range visuals {
		visuals[i].Origin[0], visuals[i].Origin[1] = lastpx, lastpy
		log.Println("Origin: ", visuals[i].Origin)
		//fmt.Println("before: ", v.rect, v.rect.Center())
		//fmt.Println("after: ", v.rect, v.rect.Center())
		origins.Vids = append(origins.Vids, visuals[i].ID)
		origins.Origins = append(origins.Origins, visuals[i].Origin)
		//HERE rpc
		lastpx += dx
		if lastpx+dx*0.5 > c.DispW {
			lastpx = c.DispW + dx*0.5
			row += 1
			lastpy = c.DispH - row*dy
		}
	}
	//CALL
	var repl int = 0
	log.Printf("calling Display.SetVisualsOrigin %v\n", origins)
	err := client.Call("Display.SetVisualsOrigin", origins, &repl)
	if err != nil {
		*reply = -1
		log.Println("Display error setting Visuals orgin: ", err)
		return err
	}
	if repl == 0 {
		reply = &repl
	} else {
		log.Println("Something happened during Display.SetVisualsOrigin ", repl)
		*reply = -1
	}

	return nil
}

//runs the given struct as an RPC service using JSON as encoding
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
