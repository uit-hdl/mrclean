package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"log"
	"math"
	"mrclean/message"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/gorilla/websocket"
)

var (
	ws  string
	id  chan int
	url string
	//mltplx is the IP:PORT for the broadcast messages
	mltplx string
	//js     bool
	wallDPI float64 = math.Sqrt(7168*7168+3072*3072) / 221.0
)

var (
	//this is to distinguish teh rpc from the noise... maybe i can get away with it
	//replicating the status, or part of it, of the visuals
	results map[int]chan *RpcRes
	//This channel is for the messages and comands and stuff
	msgChan chan interface{}
	evtChan chan Event
	rpcChan chan RpcRes
)

func init() {
	flag.StringVar(&ws, "ws", "ws://10.1.255.77:8088/ws_rpc_events", "IP:PORT of websocket server.")
	flag.StringVar(&mltplx, "mltplx", "127.0.0.1:32123", "IP:PORT for the broadcast messages")
	//flag.BoolVar(&js, "json", false, "if true print the json output, otherwise the Go sysntax")

	initID()
	//wehn a rpc result is found on the websocket we look for its ID in teh manp
	//and send the result on the channel to teh goroutine. hopefully this works...
	results = make(map[int]chan *RpcRes)

}

func main() {
	flag.Parse()
	//setting up singnal handling
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT)
	go func() {
		<-ch
		log.Println("CTRL-C")
		// shutdown()
		os.Exit(0)
	}()

	//connecting to display cloud
	conn, resp, err := websocket.DefaultDialer.Dial(ws, nil) //head)
	if err != nil {
		if err == websocket.ErrBadHandshake {
			log.Fatalf("%v\n Response: %+v\n", err, resp)
		}
		log.Fatalln(err)
	}
	log.Println("Connected.")
	msgChan, err := NetListen()
	if err != nil {
		log.Fatal(err)
	}
	rpcChan = make(chan RpcRes, 5)
	evtChan = make(chan Event, 5)
	go WSHandle(evtChan, rpcChan, conn)
	go EventHandler(evtChan)
	//go ReadWS(conn, msgchan)
	go Loop(msgChan, rpcChan, conn)
	//reading stdinput for messages
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		fmt.Println("SEND :", scanner.Text()) // Println will add back the final '\n'
		err := conn.WriteMessage(websocket.TextMessage, scanner.Bytes())
		if err != nil {
			log.Fatal(err)
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading standard input:", err)
	}

}

func initID() {
	id = make(chan int)
	count := 0
	go func() {
		for {
			id <- count
			count++
		}
	}()
}

type MessIn struct {
	*Event
	*RpcRes
}

//{"event_type": "object_property_changed", "dc_event": "1.0",
//"event_data": {"new_value": [0.05526591049184404,
//0.018044854873855876], "obj_type": "visual", "property_name": "origin", "obj_id": 38}}
type Event struct {
	Type    string                 `json:"event_type"`
	Version string                 `json:"dc_event"`
	Data    map[string]interface{} `json:"event_data"`
}

type Visual struct {
	Origin       []float64 `json:"origin,omitempty"`
	Description  string    `json:"description,omitempty"`
	Name         string    `json:"name,omitempty"`
	Type         string    `json:"type,omitempty"`
	ID           int       `json:"id,omitempty"`
	DPI          []float64 `json:"dpi,omitempty"`
	SizeDiscrete []int     `json:"size_discrete,omitempty"`
	Size         []float64 `json:"size,omitempty"`
	VNCServer    string    `json:"vnc_server,omitempty"`
	DPIHint      []float64 `json:"dpi_hint,omitempty"`
	PicGeometry  []float64 `json:"pic_geometry,omitempty"`
	URL          string    `json:"pic_url,omitempty"`
	//rect             Rectangle `json:"-"`
	message.MetaData `json:"-"`
}

//Rect returns an image.Ractangle
func (v *Visual) Rect() image.Rectangle {
	s2x := int(v.SizeDiscrete[0] / 2)
	s2y := int(v.SizeDiscrete[1] / 2)

	x0 := int(v.Origin[0]) - s2x
	y0 := int(v.Origin[1]) - s2y
	x1 := int(v.Origin[0]) + s2x
	y1 := int(v.Origin[1]) + s2y
	return image.Rect(x0, y0, x1, y1)
}

//fRect returns a port of image.Ractangle to float64
func (v *Visual) fRect() Rectangle {
	s2x := v.Size[0] / 2
	s2y := v.Size[1] / 2

	x0 := v.Origin[0] - s2x
	y0 := v.Origin[1] - s2y
	x1 := v.Origin[0] + s2x
	y1 := v.Origin[1] + s2y
	return Rect(x0, y0, x1, y1)
}

//{"origin": [0.0, 0.0],
//"description": "7x4 tiled 1024x768 displays",
//"hostname": "rocksvv.local",
//"visible": true,
//"dpi": [35.28756407422915, 35.28756407422915],
//"size": [5.1595287115033655, 2.2112265906442996],
//"id": 29,
//"sizeDiscrete": [7168, 3072],
//"name": "IfI display wall"}
type Display struct {
	Origin       []float64 `json:"origin,omitempty"`
	Description  string    `json:"description,omitempty"`
	Name         string    `json:"name,omitempty"`
	DPI          []float64 `json:"dpi,omitempty"`
	ID           int       `json:"id,omitempty"`
	SizeDiscrete []int     `json:"sizeDiscret,omitemptye"`
	Size         []float64 `json:"size,omitempty"`
	HostName     string    `json:"hostname,omitempty"`
	Visible      bool      `json:"visible,omitempty"`
	rect         Rectangle `json:"-"`
}

//fRect returns a port of image.Ractangle to float64
func (d *Display) fRect() Rectangle {
	s2x := d.Size[0] / 2
	s2y := d.Size[1] / 2

	x0 := d.Origin[0] - s2x
	y0 := d.Origin[1] - s2y
	x1 := d.Origin[0] + s2x
	y1 := d.Origin[1] + s2y
	return Rect(x0, y0, x1, y1)
}

//JSON RPC Request
type RpcReq struct {
	Version string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	ID      int         `json:"id"`
	Params  interface{} `json:"params"`
	//PosParams  []interface{}          `json:"params"`
}

//JSON RPC Result
type RpcRes struct {
	Version string `json:"jsonrpc"`
	ID      int    `json:"id"`
	//Result  interface{} `json:"result"`
	Result json.RawMessage `json:"result"`
	Error  *RpcErr         `json:"error"`
}

type RpcErr struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

///json_rpc_call('visuals_get_info', [])
func VisualsInfo() RpcReq {
	call := RpcReq{
		Version: "2.0",
		ID:      <-id,
		Method:  "visuals_get_info",
		//we need to pass a slice or the json encoding will
		// ne null and it pisses off torado
		Params: []int{}}
	log.Println("call: ", call)
	return call
}

///json_rpc_call('visual_get_info', [])
func VisualInfo(vid int) RpcReq {
	call := RpcReq{
		Version: "2.0",
		ID:      <-id,
		Method:  "visual_get_info",
		//we need to pass a slice or the json encoding will
		// ne null and it pisses off torado
		Params: []int{vid}}
	log.Println("call: ", call)
	return call
}

///json_rpc_call('visual_set_origin', [39, [0.0, 0.0]])
func SetOrigin(vid int, x, y float64) RpcReq {
	params := []interface{}{vid, []float64{x, y}}
	call := RpcReq{
		Version: "2.0",
		ID:      <-id,
		Method:  "visual_set_origin",
		Params:  params}
	return call
}

///json_rpc_call('visual_scale', [42, [2.0, 2.0]])
func SetScale(vid int, x, y float64) RpcReq {
	params := []interface{}{vid, []float64{x, y}}
	call := RpcReq{
		Version: "2.0",
		ID:      <-id,
		Method:  "visual_scale",
		Params:  params}
	return call
}

///json_rpc_call('add_visual', [{ 'name' : 'saturn', 'origin' : [0.0, 0.0], 'size_discrete' : [3536,1399], 'dpi_hint' : [70.0,70.0], 'type' : 'pic', 'pic_url' : 'http://tellus-demo.local/td-pics/saturn2.jpg', 'pic_geometry' : [3536,1399]  }])
/*"""Adds a visual to the cloud display.
  @param info dictionary with info about the visual. Keys:
      name: name of visual (optional, default '')
      description: descriptive text (optional, default '')
      origin: (x,y) initial center point of the visual in the cloud
          display
      size: (w,h) initial width and height, in m (optional)
      size_discrete: (w,h) initial width and height, in
          pixels (optional)
      dpi_hint: (x,y) initial pixel density, in dpi (optional)
      type: 'null', 'vnc', or 'pic'.

      Either size, or size_discrete and dpi_hint must be given.

      Depending on the type, there are more keys. For null,
     there are none. For vnc:
          vnc_server: vnc server url (str)
          enc_passwd: encrypted password (str)
          vnc_geometry: (w,h) width and height of framebuffer,
              in pixels
      For pic:
          pic_url: http url for picture
          pic_geometry: (w,h) width and height of pic, in pixels

  @return nothing. An event is issued when the visual has been
  added.
  """
*/
func AddVisual(v *Visual) RpcReq {
	params := []interface{}{v}
	call := RpcReq{
		Version: "2.0",
		ID:      <-id,
		Method:  "add_visual",
		Params:  params,
	}
	return call
}

///json_rpc_call('displaygroups_get_info', [])
func DisplayGroupInfo() RpcReq {
	call := RpcReq{
		Version: "2.0",
		ID:      <-id,
		Method:  "displaygroups_get_info",
		//we need to pass a slice or the json encoding will
		// ne null and it pisses off torado
		Params: []int{}}
	log.Println("call: ", call)
	return call
}

///json_rpc_call('move_selected_visuals', [[42, 39], [], [29]])
/* """moves the visuals onto either one display or one displaygroup.
   @param visual_ids list of visual ids (length > 0)
   @param display_ids list of display ids (length <= 1)
   @param displaygroup_ids list of displaygroup ids (length <= 1)
   """
*/
func MoveVisuals(vids, dids, gids []int) RpcReq {
	params := []interface{}{vids, dids, gids}
	call := RpcReq{
		Version: "2.0",
		ID:      <-id,
		Method:  "move_selected_visuals",
		Params:  params}
	return call
}

/*
def visual_set_origin(self, vid, new_origin) :
        with self._lock :
            visual = self._visuals[vid].space_tree_node
            origin_vec = Vec2D(new_origin[0], new_origin[1])
            acked(visual.setOrigin, origin_vec)
*/
func SetVisualOrigin(vid int, norigin []float64) RpcReq {
	params := []interface{}{vid, norigin}
	call := RpcReq{
		Version: "2.0",
		ID:      <-id,
		Method:  "visual_set_origin",
		Params:  params}
	return call
}

func NewPicVisual(name, description, url string, size []float64) *Visual {
	ret := &Visual{
		Origin:      []float64{0, 0}, // `json:"origin"`
		Description: description,     //  string    `json:"description"`
		Name:        name,            //        string    `json:"name"`
		Type:        "pic",           //         string    `json:"type"`
		//ID           int       `json:"id"`
		//DPI:           []float64{70,70}// `json:"dpi"`
		SizeDiscrete: []int{int(size[0]), int(size[1])}, //    `json:"sizeDiscrete"`
		//Size:         size,                              //         []float64 `json:"size"`
		//VNCServer    string    `json:"vnc_server"`
		DPIHint:     []float64{wallDPI, wallDPI}, // `josn:"dpi_hint"`
		PicGeometry: size,                        //  []float64 `json:"pic_geometry"`
		URL:         url,                         //string    `json:"pic_url"`

	}
	return ret
}

////TODO finish this
//func VisualsInfo() []Visual {
//	ret := make([]Visual, 0, 10)
//	call := RpcReq{
//		Version: "2.0",
//		ID:      <-id,
//		Method:  "visuals_get_info",
//		//we need to pass a slice or the json encoding will
//		// ne null and it pisses off torado
//		Params: []int{}}
//	//log.Println("call: ", call)
//	return ret
//}

func NetListen() (chan interface{}, error) {
	log.Printf("attempting to listen to %v\n", message.LB)
	conn, err := net.ListenUDP("udp4", message.LB)
	//conn, err := net.ListenMulticastUDP("udp4", nil, message.Mcast)
	//conn, err := message.JoinMcast(message.Mcast.IP)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("listening on ", conn.LocalAddr())
	//ifc, err := message.McastInterfaces()
	//if err != nil {
	//	log.Fatal(err)
	//}
	//log.Println(ifc)
	//err = conn.SetReadBuffer(1 << 12) //4Kb
	//if err != nil {
	//	log.Fatal(err)
	//}

	//	conn, err := net.Dial("udp", message.Mcast)
	//	if err != nil {
	//		log.Println("Error dialing ", err)
	//	}
	//	//log.Println("conn", conn)
	//	bcastconn, ok := conn.(*net.UDPConn)
	//	if !ok {
	//		log.Println("bcastconn not UDP")
	//	}
	//	//big buffer out
	//	if err := bcastconn.SetReadBuffer(1 << 21); err != nil {
	//		log.Println("error setting read buffer, ", err)
	//		return nil, err
	//	}
	//wall, err := ecs.DialSenderJF(net, addr)
	//conn, err := net.Dial(network, addr)
	//if err != nil {
	//	log.Printf("Error Dialing: %+v\n", err)
	//	return nil, err
	//}
	//enc := json.NewDecoder(conn)
	ch := make(chan interface{}, 5) //buffer some just in case
	buff := make([]byte, 1<<12)
	go func() {
		for {
			//buff, err := json.Unmarshal(i)
			//if err != nil {
			//	log.Fatal(err)
			//}
			//log.Println("Sending ", string(buff))
			//msg := &ecs.InputMessage{}
			//err = json.Unmarshal(buff, msg)

			//if err != nil {
			//	log.Println()

			//}
			//log.Printf("decoding %v\n", msg)
			//i := &message.ImageData{} //&ecs.Message{}
			i := &message.InMessage{} //&ecs.Message{}
			n, err := conn.Read(buff)
			if err != nil {
				log.Fatal(err)
			}
			//err = enc.Decode(i)
			err = json.Unmarshal(buff[:n], i)
			if err != nil {
				log.Println(err)
				continue
			}
			switch i.Header {
			case message.ImageMsg:
				img := &message.ImageData{}
				err = json.Unmarshal(i.Content, img)
				if err != nil {
					log.Println(err)
					continue
				}
				ch <- *img
			case message.SortMsg:
				srt := &message.SortData{}
				err = json.Unmarshal(i.Content, srt)
				if err != nil {
					log.Println(err)
					continue
				}
				ch <- *srt
			}

		}
	}()
	return ch, nil
}

func ReadWSText(conn *websocket.Conn) {
	for {
		//evt := new(interface{})
		//evt := new(Event)
		_, buff, err := conn.ReadMessage()
		//err := conn.ReadJSON(evt)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("RECV:", string(buff))
		//switch {
		//case evt.Event != nil:
		//	fmt.Printf("RECV: %+v\n", *evt.Event)
		//case evt.RpcRes != nil:
		//	fmt.Printf("RECV: %+v\n", *evt.RpcRes)
		//	if evt.RpcRes.Error != nil {
		//		fmt.Printf("ERROR: %+v\n", *evt.RpcRes.Error)
		//	}
		//	if evt.RpcRes.Error.Data != nil {
		//		fmt.Printf("DATA: %+v\n", *evt.RpcRes.Error.Data)
		//	}
		//}

	}
}

//ReadWS reads the websocket and parse the json into events or rpc results. The results
//are sent on a channel.
func ReadWS(conn *websocket.Conn, ch chan interface{}) {
	for {
		//evt := new(interface{})
		evt := new(MessIn)
		//evt := new(Event)
		//_, buff, err := conn.ReadMessage()
		err := conn.ReadJSON(evt)
		if err != nil {
			log.Fatal(err)
		}
		//fmt.Println(string(buff))
		switch {
		case evt.Event != nil:
			fmt.Printf("RECV: %#v\n", *evt.Event)
		case evt.RpcRes != nil:
			//fmt.Printf("RECV: %#v\n", *evt.RpcRes)
			//TODO get the result here parse it and pass it to the loop
			//so it can handle the results of the rpc call
			ch <- *evt.RpcRes
			if evt.RpcRes.Error != nil {
				fmt.Printf("ERROR: %#v\n", *evt.RpcRes.Error)
				if evt.RpcRes.Error.Data != nil {
					fmt.Printf("DATA: %#v\n", evt.RpcRes.Error.Data)
				}
			}
		}

	}
}

//hndles events, for now just print
func EventHandler(ch chan Event) {
	for evt := range ch {
		fmt.Printf("RECV: %#v\n", evt)
	}
}

//WSHandle is a websocket handler, it uses two channels to communcate events and
//RPC from e to the websocket
func WSHandle(evtChan chan Event, rpcChan chan RpcRes, conn *websocket.Conn) {
	for {
		//evt := new(interface{})
		evt := new(MessIn)
		//evt := new(Event)
		//_, buff, err := conn.ReadMessage()
		err := conn.ReadJSON(evt)
		if err != nil {
			log.Fatal(err)
		}
		//fmt.Println(string(buff))
		switch {
		case evt.Event != nil:
			//fmt.Printf("RECV: %#v\n", *evt.Event)
			select {
			case evtChan <- *evt.Event:
			default:
				log.Println("Event channel full discarding: ", *evt.Event)
			}
		case evt.RpcRes != nil:
			//fmt.Printf("RECV: %#v\n", *evt.RpcRes)
			//fmt.Println("DATA: ", string(evt.RpcRes.Result))
			//TODO get the result here parse it and pass it to the loop
			//so it can handle the results of the rpc call
			if evt.RpcRes.Error != nil {
				fmt.Printf("ERROR: %#v\n", *evt.RpcRes.Error)
				if evt.RpcRes.Error.Data != nil {
					fmt.Printf("DATA: %#v\n", evt.RpcRes.Error.Data)
				}
			}
			select {
			case rpcChan <- *evt.RpcRes:
			default:
				log.Println("RPC respose  channel full discarding: ", *evt.RpcRes)
			}
		}
	}
}

//TODO send on th channel the result of the rpc do we cna get the display cloud id
//of the visual.
func Loop(in chan interface{}, rpc chan RpcRes, ws *websocket.Conn) {
	//pendingVis rpc calls waiting for a response containing a visual ID
	//pendingVis := make(map[int]RpcReq)
	//map of visuals
	vismap := make(map[string]*Visual)
	log.Println("getting the display wall info")
	dwinfo := DisplayGroupInfo()
	buff, err := json.Marshal(dwinfo)
	if err != nil {
		log.Fatal("Error marshaling display wall info req: ", err)
	}
	err = ws.WriteMessage(websocket.TextMessage, buff)
	if err != nil {
		log.Println("Error sending request: ", err)
	}
	log.Println("SENT: ", string(buff))
	log.Println("waiting for rpc resp on channel...")
	res := <-rpc
	log.Println("got it!")
	//check for error
	if res.Error != nil {
		log.Fatalln(res.Error)
	}
	//decode the result
	dwList := []Display{}
	err = json.Unmarshal(res.Result, &dwList)
	if err != nil {
		log.Println("Error decoding Rpc result ", err)
	}
	//	log.Println(dwList)

	display := dwList[0]
	display.rect = display.fRect()
	log.Printf("%+v\n", display)
	log.Println("now loop")

	for {
		msgin := <-in
		log.Println("RECV: ", msgin)
		switch msgin := msgin.(type) {
		case message.ImageData:
			// NewPicVisual(name, description, url string, size []float64)
			size := []float64{float64(msgin.Size[0]), float64(msgin.Size[1])}
			vis := NewPicVisual(msgin.Name, msgin.MetaData.String(), msgin.URL, size)
			req := AddVisual(vis)
			buff, err := json.Marshal(req)
			if err != nil {
				log.Println("Error sending: ", err)
				continue
			}
			//err := ws.WriteJSON(req)
			err = ws.WriteMessage(websocket.TextMessage, buff)
			if err != nil {
				log.Println("Error sending request: ", err)
			}
			log.Println("SENT: ", string(buff))
			log.Println("waiting for rpc resp on channel...")
			res := <-rpc
			log.Println("got it!")
			//check for error
			if res.Error != nil {
				log.Println(res.Error)
				continue
			}
			//decode the result
			id := new(int)
			err = json.Unmarshal(res.Result, id)
			if err != nil {
				log.Println("Error decoding Rpc result ", err)
			}
			////////////////////////////////////
			//get all the frigging info witht the ID of the DC
			req = VisualInfo(*id)
			buff, err = json.Marshal(req)
			if err != nil {
				log.Println("Error sending: ", err)
				continue
			}
			//err := ws.WriteJSON(req)
			err = ws.WriteMessage(websocket.TextMessage, buff)
			if err != nil {
				log.Println("Error sending request: ", err)
			}
			log.Println("SENT: ", string(buff))
			log.Println("waiting for rpc resp on channel...")
			res = <-rpc
			log.Println("got it!")
			//check for error
			if res.Error != nil {
				log.Println(res.Error)
				continue
			}
			//decode the result
			vis = &Visual{}
			err = json.Unmarshal(res.Result, vis)
			if err != nil {
				log.Println("Error decoding Rpc result ", err)
			}

			////////////////////////////////////
			//we got the result to the request, the id of the visual
			//so we need to get the visual from the request
			//ii := req.Params.([]interface{})
			//vis = ii[0].(Visual)
			//vis.ID = *id
			//vis.rect = vis.fRect()
			vis.MetaData = msgin.MetaData
			//vis.Description = msgin.MetaData.String()

			vismap[vis.Name] = vis
			log.Printf("%+v\n", vismap)
			//pendingVis[req.ID] = req
		case message.SortData:
			log.Println("Sorting...")
			vislist, err := FilterVisuals("mrc_", ws, rpc)
			if err != nil {
				log.Println("Error getting vis list: ", err)
			}

			reqs, err := SortVisuals(vislist, &display, msgin.Order)
			if err != nil {
				log.Println(err)
				continue
			}
			log.Println("sorted locally, sending rpcs")

			for _, req := range reqs {
				buff, err := json.Marshal(req)
				if err != nil {
					log.Println("Error sending: ", err)
					continue
				}
				//err := ws.WriteJSON(req)
				err = ws.WriteMessage(websocket.TextMessage, buff)
				if err != nil {
					log.Println("Error sending request: ", err)
				}
				log.Println("SENT: ", string(buff))
				//TODO see if we can avoid waiting for the response
				//by omittin the req id
				log.Println("waiting for rpc resp on channel...")
				res := <-rpc
				log.Println("got it!")
				//check for error
				if res.Error != nil {
					log.Println("Error sorting on Display, ", res.Error)
				}
			}
			log.Println("...done.")

		}
	}
}

func FilterVisuals(prefix string, ws *websocket.Conn, rpc chan RpcRes) ([]*Visual, error) {
	// NewPicVisual(name, description, url string, size []float64)
	req := VisualsInfo()
	buff, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("Error marshalling: ", err)
	}
	//err := ws.WriteJSON(req)
	err = ws.WriteMessage(websocket.TextMessage, buff)
	if err != nil {
		log.Println("Error sending request: ", err)
	}
	log.Println("SENT: ", string(buff))
	log.Println("waiting for rpc resp on channel...")
	res := <-rpc
	log.Println("got it!")
	//check for error
	if res.Error != nil {
		return nil, fmt.Errorf("Error in response: ", err)
	}
	//decode the result
	var ret, tmp []*Visual
	err = json.Unmarshal(res.Result, &tmp)
	if err != nil {
		return nil, fmt.Errorf("Error decoding Rpc result: ", err)
	}
	//filter the visuals with our prefix
	log.Printf("Received %d visuals\n", len(tmp))
	for _, vis := range tmp {
		if strings.HasPrefix(vis.Name, prefix) {
			err := vis.MetaData.Parse(vis.Description)
			if err != nil {
				log.Println(err)
				continue
			}

			ret = append(ret, vis)
		}
	}
	log.Printf("Sorting %d visuals\n", len(ret))

	return ret, nil
}

//func SortImages(imgs map[string]*Visual, disp Display) []RpcReq {
//	//get the bigger image, it will be reference block
//	//for the sorting
//	maxr := Rectangle{}
//	for _, v := range imgs {
//		maxr = maxr.Union(v.rect)
//	}
//	var (
//		margin float64 = 0.05 //5 cm
//		row    float64 = 1
//		ret    []RpcReq
//	)
//	dx, dy := maxr.Dx()+margin, maxr.Dy()+margin
//	lastpos := Point{Y: disp.Size[1] - dy}
//	for _, v := range imgs {
//		v.rect.Min = lastpos
//		ret = append(ret,
//			SetVisualOrigin(v.ID, v.rect.Center().Slice()))
//		//HERE rpc
//		lastpos.X += dx
//		if lastpos.X+dx > disp.Size[0] {
//			lastpos.X = 0
//			row += 1
//			lastpos.Y = disp.Size[1] - row*dy
//		}
//	}
//	return ret
//}

//TODO a sort function to sort the images in some fashion on the display
//decide different sortings, use metadata or the image name or both.
//for example split the path (/) and split the name (_) of the image to get more info.
//preudocode?
/*
if we dont have a display get the display from displaycloud
get visuals from displaycloud
filter visuals to geto only the mr clean ones using the name prefix
sort them on screen.
*/
/*
func JoinMcast(group *net.UDPAddr) (*net.UDPConn, error) {
	conn, err := net.ListenPacket("udp4", "0.0.0.0:1024")
	if err != nil {
		log.Fatal(err)
	}
	//p := ipv4.NewPacketConn(conn)
	//en, err := p.MulticastInterface()
	//if err != nil {
	//	log.Println("multicast interface")
	//	return nil, err
	//}
	//if err := p.JoinGroup(nil, &net.UDPAddr{IP: group.IP}); err != nil {
	//	// error handling
	//}
	//err = conn.SetReadBuffer(1 << 12) //4Kb
	//if err != nil {
	//	log.Fatal(err)
	//}
	return conn.(*net.UDPConn), nil

}
*/
