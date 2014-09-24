package displaycloud

import (
	"encoding/json"
	"fmt"
	"image"
	"log"
	"math"

	"github.com/UniversityofTromso/mrclean"
	"github.com/gorilla/websocket"
)

var (
	id      chan int
	wallDPI float64 = math.Sqrt(7168*7168+3072*3072) / 221.0
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	initID()

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
	Origin          []float64 `json:"origin,omitempty"`
	Description     string    `json:"description,omitempty"`
	Name            string    `json:"name,omitempty"`
	Type            string    `json:"type,omitempty"`
	ID              int       `json:"id,omitempty"`
	DPI             []float64 `json:"dpi,omitempty"`
	SizeDiscreteBug []int     `json:"sizeDiscrete,omitempty"`
	SizeDiscrete    []int     `json:"size_discrete,omitempty"`
	Size            []float64 `json:"size,omitempty"`
	VNCServer       string    `json:"vnc_server,omitempty"`
	DPIHint         []float64 `json:"dpi_hint,omitempty"`
	PicGeometry     []float64 `json:"pic_geometry,omitempty"`
	URL             string    `json:"pic_url,omitempty"`
	//rect             Rectangle `json:"-"`
	mrclean.MetaData `json:"-"`
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

////fRect returns a port of image.Ractangle to float64
//func (v *Visual) fRect() Rectangle {
//	s2x := v.Size[0] / 2
//	s2y := v.Size[1] / 2
//
//	x0 := v.Origin[0] - s2x
//	y0 := v.Origin[1] - s2y
//	x1 := v.Origin[0] + s2x
//	y1 := v.Origin[1] + s2y
//	return Rect(x0, y0, x1, y1)
//}

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
	SizeDiscrete []int     `json:"sizeDiscret,omitempty"`
	Size         []float64 `json:"size,omitempty"`
	HostName     string    `json:"hostname,omitempty"`
	Visible      bool      `json:"visible,omitempty"`
	//rect         Rectangle `json:"-"`
}

//fRect returns a port of image.Ractangle to float64
//func (d *Display) fRect() Rectangle {
//	s2x := d.Size[0] / 2
//	s2y := d.Size[1] / 2
//
//	x0 := d.Origin[0] - s2x
//	y0 := d.Origin[1] - s2y
//	x1 := d.Origin[0] + s2x
//	y1 := d.Origin[1] + s2y
//	return Rect(x0, y0, x1, y1)
//}

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
		// be null and it pisses off torado
		Params: []int{}}
	log.Println("call: ", call)
	return call
}

///json_rpc_call('displays_get_info', [])
func DisplaysInfo() RpcReq {
	call := RpcReq{
		Version: "2.0",
		ID:      <-id,
		Method:  "displays_get_info",
		//we need to pass a slice or the json encoding will
		// be null and it pisses off torado
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

//hndles events, for now just print
func eventHandler(ch chan Event) {
	for evt := range ch {
		fmt.Printf("RECV: %#v\n", evt)
	}
}

type Client struct {
	conn    *websocket.Conn
	evtChan chan Event
	rpcChan chan RpcRes
	Display Display
}

func Dial(url string) (*Client, error) {
	conn, resp, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		if err == websocket.ErrBadHandshake {
			return nil, fmt.Errorf("%v\n Response: %+v\n", err, resp)
		}
	}
	log.Println("Connected to DisplayCloud.")
	cli := &Client{
		conn:    conn,
		rpcChan: make(chan RpcRes, 10),
		evtChan: make(chan Event, 10),
	}
	// start the asynch goroutine for the RPC response
	go eventHandler(cli.evtChan)
	go cli.handleConn()
	err = cli.displayInfo()
	if err != nil {
		//without display info makes no sense ot proceed
		log.Fatal(err)
	}
	//cli.Display = *d
	return cli, nil
}
func (cli *Client) handleConn() {
	for {
		//evt := new(interface{})
		evt := new(MessIn)
		//evt := new(Event)
		//_, buff, err := conn.ReadMessage()
		err := cli.conn.ReadJSON(evt)
		if err != nil {
			log.Fatal(err)
		}
		//fmt.Println(string(buff))
		switch {
		case evt.Event != nil:
			if cli.evtChan != nil {
				//fmt.Printf("RECV: %#v\n", *evt.Event)
				select {
				case cli.evtChan <- *evt.Event:
				default:
					log.Println("Event channel full discarding: ", *evt.Event)
				}
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
			case cli.rpcChan <- *evt.RpcRes:
			default:
				log.Println("RPC respose  channel full discarding: ", *evt.RpcRes)
			}
		}
	}
}

func (cli *Client) displayInfo() error {
	log.Println("getting the display wall info")
	dwinfo := DisplaysInfo()
	buff, err := json.Marshal(dwinfo)
	if err != nil {
		//log.Fatal("Error marshaling display wall info req: ", err)
		return err
	}
	err = cli.conn.WriteMessage(websocket.TextMessage, buff)
	if err != nil {
		//log.Println("Error sending request: ", err)
		return err
	}
	log.Println("SENT: ", string(buff))
	log.Println("waiting for rpc resp on channel...")
	res := <-cli.rpcChan
	log.Println("got it!")
	//check for error
	if res.Error != nil {
		return fmt.Errorf("%+v", res.Error)
		//log.Fatalln(res.Error)
	}
	//decode the result
	dwList := []Display{}
	err = json.Unmarshal(res.Result, &dwList)
	if err != nil {
		return err
		//log.Println("Error decoding Rpc result ", err)
	}
	//	log.Println(dwList)

	cli.Display = dwList[0]
	//cli.Display.rect = cli.Display.fRect()
	log.Printf("Received: %+v\n", dwList[0])
	//log.Printf("Set: %+v\n", cli.Display)
	//cli.Display = display //dwList[0]
	//return &dwList[0], nil
	return nil
}

func (cli *Client) AddVisual(vis mrclean.Visual) (*Visual, error) {
	nvis := &Visual{
		Origin:       []float64{0, 0},
		Name:         vis.Name,
		Type:         "pic",
		SizeDiscrete: []int{vis.Rectangle.Dx(), vis.Rectangle.Dy()},
		DPIHint:      []float64{wallDPI, wallDPI},
		PicGeometry:  []float64{float64(vis.Rectangle.Dx()), float64(vis.Rectangle.Dy())},
		URL:          vis.URL,
	}
	req := AddVisual(nvis)
	buff, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("Error %v marshaling %+v ", err, req)
	}
	//err := ws.WriteJSON(req)
	err = cli.conn.WriteMessage(websocket.TextMessage, buff)
	if err != nil {
		return nil, fmt.Errorf("Error %v sending %+v ", err, req)
		//log.Println("Error sending request: ", err)
	}
	log.Println("SENT: ", string(buff))
	log.Println("waiting for rpc resp on channel...")
	res := <-cli.rpcChan
	log.Println("got it!")
	//check for error
	if res.Error != nil {
		return nil, fmt.Errorf("%+v", res.Error)
	}
	//decode the result
	id := new(int)
	err = json.Unmarshal(res.Result, id)
	if err != nil {
		return nil, fmt.Errorf("Error decoding Rpc result %v", err)
	}
	////////////////////////////////////
	//get all the info witht the ID of the DC
	req = VisualInfo(*id)
	buff, err = json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("Error %v marshaling %+v ", err, req)
	}
	//err := ws.WriteJSON(req)
	err = cli.conn.WriteMessage(websocket.TextMessage, buff)
	if err != nil {
		return nil, fmt.Errorf("Error %v sending %+v ", err, req)
	}
	log.Println("SENT: ", string(buff))
	log.Println("waiting for rpc resp on channel...")
	res = <-cli.rpcChan
	log.Println("got it!")
	//check for error
	if res.Error != nil {
		return nil, fmt.Errorf("Error: %+v", res.Error)
	}
	log.Printf("Result: %s", string(res.Result))
	//decode the result
	vis2 := Visual{}
	err = json.Unmarshal(res.Result, &vis2)
	if err != nil {
		return nil, fmt.Errorf("Error decoding Rpc result %v", err)
	}

	////////////////////////////////////
	//we got the result to the request, the id of the visual
	//so we need to get the visual from the request
	//ii := req.Params.([]interface{})
	//vis = ii[0].(Visual)
	//vis.ID = *id
	//vis.rect = vis.fRect()
	//vis.MetaData = msgin.MetaData
	//vis.Description = msgin.MetaData.String()

	//vismap[vis.Name] = vis
	log.Printf("Visual: %+v\n", vis2)
	//pendingVis[req.ID] = req

	return &vis2, nil

}

func (cli *Client) SetVisualsOrigin(vids []int, poss [][]float64) error {
	if len(vids) != len(poss) {
		return fmt.Errorf("Mismatched length in args")
	}
	for i := range vids {
		req := SetVisualOrigin(vids[i], poss[i])
		buff, err := json.Marshal(req)
		if err != nil {
			return fmt.Errorf("Error sending: %v", err)
		}
		//err := ws.WriteJSON(req)
		err = cli.conn.WriteMessage(websocket.TextMessage, buff)
		if err != nil {
			log.Println("Error sending reqrest: ", err)
		}
		log.Println("SENT: ", string(buff))
		//TODO see if we can avoid waiting for the response
		//by omittin the req id
		log.Println("waiting for rpc resp on channel...")
		res := <-cli.rpcChan
		log.Println("got it!")
		//check for error
		if res.Error != nil {
			return fmt.Errorf("Error sorting on Display, %v", res.Error)
		}
	}
	return nil
}

//
//func (cli *Client) DisplayInfo() error {
//
//	req := DisplayGroupInfo()
//	buff, err := json.Marshal(req)
//	if err != nil {
//		return fmt.Errorf("Error sending: %v", err)
//	}
//	//err := ws.WriteJSON(req)
//	err = cli.conn.WriteMessage(websocket.TextMessage, buff)
//	if err != nil {
//		return fmt.Errorf("Error sending reqrest: ", err)
//	}
//	log.Println("SENT: ", string(buff))
//	//TODO see if we can avoid waiting for the response
//	//by omittin the req id
//	log.Println("waiting for rpc resp on channel...")
//	res := <-cli.rpcChan
//	log.Println("got it!")
//	//check for error
//	if res.Error != nil {
//		return fmt.Errorf("Error sorting on Display, %v", res.Error)
//	}
//	//decode the result
//	dwList := []Display{}
//	err = json.Unmarshal(res.Result, &dwList)
//	if err != nil {
//		return fmt.Errorf("Error decoding Rpc result ", err)
//	}
//	//	log.Println(dwList)
//
//	cli.Display = dwList[0]
//	cli.Display.rect = cli.Display.fRect()
//	log.Printf("%+v\n", cli.Display)
//
//	return nil
//}
