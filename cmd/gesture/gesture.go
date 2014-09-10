package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/rpc"
	"net/rpc/jsonrpc"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"

	glm "github.com/folago/googlmath"

	"github.com/UniversityofTromso/mrclean"
	"github.com/folago/leap"
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
	//netconn is the transport protocol for the connection
	netconn string

	//period of polling
	//T time.Duration = 16 * time.Millisecond
	//channel for gestures fomr leapmotion
	out chan []leap.Gesture
)

func init() {
	runtime.LockOSThread()
	flag.StringVar(&corerpc,
		"corerpc", mrclean.CoreAddr,
		"IP:PORT of the Core RPC server")
	flag.StringVar(&configfile,
		"configfile", "config.json", "Configuration file for Mr. Clean gestures")
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
	client, err = jsonrpc.Dial(netconn, corerpc)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	//fmt.Println("Exanmple of message: ", string(buff))
	go StdInput()
	go LeapSend()

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

func StdInput() {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("cmd>")
	var ret int

	for scanner.Scan() {
		fmt.Println("SEND :", scanner.Text()) // Println will add back the final '\n'
		err := client.Call("Core.Sort", scanner.Text(), &ret)
		if err != nil {
			log.Println(err)
		}
		if ret == -1 {
			log.Printf("something wrong in the sorting")
		}
		fmt.Print("cmd>")
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading standard input:", err)
	}

}

func LeapSend() {
	out = make(chan []leap.Gesture, 10)
	//leap motion setup
	ldev, err := leap.Dial(leap.WSURL)
	//log.Println(err, ldev)
	if err != nil {
		log.Fatal(err)
	}
	ldev.GestEnable(true)
	go GestureSender(out, ldev)
	for gl := range out {
		//fmt.Printf("id: %d, type: %s frames: %d ", gl[0].ID, gl[0].Type, len(gl))
		//dur := gl[0].Duration
		for _, g := range gl {
			switch g.Type {
			case "circle":
				x := g.Normal.Dot(glm.Vector3{0, 0, -1})
				var clockwise bool
				layers := strings.Split(config["layers"], "/")
				if x >= 0 {
					clockwise = true
					shift(layers, clockwise)

				} else {
					shift(layers, clockwise)
				}
				sort := strings.Join(layers, "/")
				fmt.Println("SEND :", sort)
				var ret int
				err := client.Call("Core.Sort", sort, &ret)
				if err != nil {
					log.Println(err)
				}
				if ret == -1 {
					log.Printf("something wrong in the sorting")
				}
				config["layers"] = sort
				log.Printf("id: %d, type: %s progress: %f clockwise: %v \n", g.ID, g.Type, g.Progress, clockwise)
			case "swipe":
				log.Printf("id: %d, type: %s speed: %f \n", g.ID, g.Type, g.Speed)
				layers := strings.Split(config["layers"], "/")
				shuffle(layers)
				group := strings.Join(layers, "/")
				fmt.Println("SEND :", group)
				//group := layers[rand.Intn(len(layers))]
				//fmt.Println("SEND :", group)
				var ret int
				err := client.Call("Core.Group", group, &ret)
				if err != nil {
					log.Println(err)
				}
				if ret == -1 {
					log.Printf("something wrong in the sorting")
				}
			case "screenTap":
				log.Printf("id: %d, type: %s\n", g.ID, g.Type)
			case "keyTap":
				log.Printf("id: %d, type: %s\n", g.ID, g.Type)
			}
			//dur += g.Duration
			//if g.Type == "circle" {
			//	fmt.Printf("%v", g.Normal)
			//}
		}
		//fmt.Printf("duration: %v\n", dur)
		//str := fmt.Sprintf("id: %d, state: %s type: %s", gl[0].ID, gl[0].State gl[0].Type)
		//buff := bytes.NewBufferString(str)
		//for _, g := range gl[1:] {
		//	_, err := buff.WriteString(fmt.Sprintf("id: %d, state: %s ", g.ID, g.State))
		//	if err != nil {
		//		log.Fatal(err)
		//	}
		//}
		//fmt.Printf("%s\n\n\n", buff)
	}
}

//TODO this is soooo ineficient so fix it eventually
func GestureSender(ch chan []leap.Gesture, ld *leap.Device) {
	//gmap := make(map[int]leap.Gesture)
	for frame := range ld.Frames {
		//fmt.Printf("%+v\n", frame.Timestamp)

		if len(frame.Gestures) == 0 {
			//fmt.Printf("No gestures\n")
			continue
		}
		var gslice []leap.Gesture
		//get a gest by id
		for _, g := range frame.Gestures {
			//fmt.Printf("%+v\n", g)
			if g.State == "stop" {
				gslice = append(gslice, g)
			}
			//fmt.Printf("%+v\n", g)
		}
		if len(gslice) > 0 {
			ch <- gslice
		}
	}
	//log.Printf("%v  Radius %f\n", handmove, gest.SphereRadius)
	//ch <- ecs.InputMessage{HandMove: handmove}
}

//shuffles the metadate randomly
func shuffle(slice []string) { //[]string {
	if slice == nil {
		return //slice
	}
	for i := range slice {
		j := rand.Intn(i + 1)
		slice[i], slice[j] = slice[j], slice[i]
	}
	return //slice
}

//shift the metadata order and wraps around
func shift(slice []string, left bool) { // []string {
	if slice == nil {
		return //slice
	}
	temp := make([]string, len(slice))
	copy(temp, slice)
	if left {
		for i, v := range temp[1:] {
			slice[i] = v
		}
		slice[len(slice)-1] = temp[0]
	}
	if !left {
		for i, v := range temp[:len(slice)-1] {
			slice[i+1] = v
		}
		slice[0] = temp[len(slice)-1]
	}
	return
}

func Map(ch chan leap.Gesture) chan leap.Gesture {
	return ch
}

////init glfe and gamepad, loop foreverrrrrrrr
//func Pad() {
//err = glfw.Init()
//	if err != nil {
//		log.Println("glfw.Init failed: ", err)
//		return
//	}
//	defer glfw.Terminate()
//	win.W, err = glfw.CreateWindow(800, 800, "Testing", nil, nil)
//	if err != nil {
//		panic(err)
//	}
//	win.EnablePadPressChan()
//	win.EnableKeyPressChan()
//	win.HookEvents()
//	//win.EnablePadReleaseChan()
//	j := glfw.Joystick(0)
//	if glfw.JoystickPresent(j) {
//		//name, err := j.Name()
//		name := glfw.GetJoystickName(j)
//		//if err != nil {
//		//	log.Println(err)
//		//}
//		log.Println("Joystick", name)
//	} else {
//		log.Println("No Joystick found.")
//	}
//	ticker := time.Tick(T)
//	for {
//		<-ticker
//		win.PollEvents()
//		win.PollGamePad(j)
//		select {
//		default: //no events, see ya next loop
//		case padpress := <-win.PadPressChan:
//			log.Println(padpress)
//		case keypress := <-win.KeyPressChan:
//			log.Println(keypress)
//		}
//	}
////}
