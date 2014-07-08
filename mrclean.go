package mrclean

import (
	"encoding/json"
	"fmt"
	"gui/we"
	"image"
	"log"
	"net"
	"strings"

	"code.google.com/p/go.net/ipv4"
)

//messgae type
const (
	MInput = "input"
	MCmd   = "command"
)

const (
	ImageMsg = "ImageData"
	SortMsg  = "SortData"
)

//ports for the components
const (
	CoreAddr      string = ":32123"
	ChronicleAddr string = ":32124"
	DisplayAddr   string = ":32125"
)

var (
	LB *net.UDPAddr = &net.UDPAddr{

		IP:   net.ParseIP("127.0.0.1"),
		Port: 32123,
	}
	Mcast *net.UDPAddr = &net.UDPAddr{

		IP:   net.ParseIP("224.0.0.3"),
		Port: 32123,
	}
	//Mcast          string = "224.0.0.3:32123"
	DisplayCloudWS = "ws://10.1.255.77:8088/ws_rpc_events"
)

type OutMessage struct {
	Header  string      //so fat just a string indicating the kind of message
	Content interface{} //json.RawMessage
}

type InMessage struct {
	Header  string //so fat just a string indicating the kind of message
	Content json.RawMessage
}

type InputMessage struct {
	Drag       *we.MouseDrag
	Press      *we.MousePress
	Release    *we.MouseRelease
	Move       *we.MouseMove
	PadPress   *we.PadPress
	PadRelease *we.PadRelease
	PadMove    *we.PadMove
	PadDrag    *we.PadDrag
	AxisMove   *we.AxisMove
	HandMove   *we.HandMove
}

type CmdMessage struct {
	NewImage *ImageData //*string
	Sort     *SortData
}

type ImageData struct {
	Name string
	Size [2]int
	URL  string
	//Meta []string
	MetaData
}

type MetaData struct {
	Task      string
	Approach  string
	Iteration string //int
	Method    string
}

func (m *MetaData) Parse(info string) error {
	str := strings.Split(info, "/")
	if len(str) < 4 {
		return fmt.Errorf("Not enough info for metadata: %s", info)
	}
	m.Task = str[0]
	m.Approach = str[1]
	m.Iteration = str[2]
	m.Method = str[3]
	return nil
}

func (m *MetaData) String() string {
	return fmt.Sprintf("%s/%s/%s/%s", m.Task, m.Approach, m.Iteration, m.Method)
}

//sorting interface stuff
type Images []*ImageData

//partial implementationn of sort.Interface the rest is implementes
//in the other types
func (i Images) Len() int      { return len(i) }
func (i Images) Swap(p, q int) { i[p], i[q] = i[q], i[p] }

//sort by name
type ByName struct{ Images }

func (n ByName) Less(i, j int) bool { return n.Images[i].Name < n.Images[j].Name }

//sort by Iteration
type ByIteration struct{ Images }

func (n ByIteration) Less(i, j int) bool { return n.Images[i].Iteration < n.Images[j].Iteration }

//sort by Approach
type ByApproach struct{ Images }

func (n ByApproach) Less(i, j int) bool { return n.Images[i].Approach < n.Images[j].Approach }

//sort by Method
type ByMethod struct{ Images }

func (n ByMethod) Less(i, j int) bool { return n.Images[i].Method < n.Images[j].Method }

type SortData struct {
	Order []string //int
}

func JoinMcast(group net.IP) (*net.UDPConn, error) {
	conn, err := net.ListenPacket("udp4", Mcast.String())
	if err != nil {
		return nil, err
	}
	p := ipv4.NewPacketConn(conn)
	//en, err := p.MulticastInterface()
	//if err != nil {
	//	log.Println("multicast interface")
	//	return nil, err
	//}
	ifcs, err := McastInterfaces()
	if err != nil {
		return nil, err
	}

	for _, ifc := range ifcs {
		if err := p.JoinGroup(&ifc, &net.UDPAddr{IP: group}); err != nil {
			return nil, err
		}
	}
	//err = conn.SetReadBuffer(1 << 12) //4Kb
	//if err != nil {
	//	log.Fatal(err)
	//}
	log.Println("Connected to multicast: ")
	udp, ok := conn.(*net.UDPConn)
	if !ok {
		return nil, fmt.Errorf("Cannot convert to *net.UDPConn")
	}
	return udp, nil

}

func McastInterfaces() ([]net.Interface, error) {
	tt, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	ret := make([]net.Interface, 0, 1)
	for _, t := range tt {
		if t.Flags&(net.FlagUp) != 0 && (net.FlagMulticast) != 0 {
			ret = append(ret, t)
		}
	}
	if len(ret) == 0 {
		return nil, fmt.Errorf("Cannot find a multicast interface")
	}
	return ret, nil
}

// Visual represents a visual object to be displaied.
type Visual struct {
	//The id of the visual
	ID int
	// The name of the visual
	Name string
	//The rectangle holding proportion and size for the visual in pixels
	Rectangle image.Rectangle
	//The url specifiyng where to find the hvisual
	URL string
	//Metadata associated with the visual
	Meta []string
	//Origin is the handle to move teh Visual
	Origin []float64
	//Size represent the size on screen
	Size []float64
}

// A Gestue is a gesture performed by a user.
type Gesture struct {
	ID    int
	Name  string
	Param []string
}

type VisualOrigins struct {
	Vids    []int
	Origins [][]float64
}
