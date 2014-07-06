//Watcher is a flie warcher for mr clean, it watches files created in a directory tree and
//when a new file is created watcher sends a message to mr clean
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/folago/mrclean"
	"github.com/fsnotify/fsnotify"
)

//flags
var (
	//watch contains the path to watch for files to be added or changed
	watch string
	//mltplx is the IP:PORT we are going to connect to and mltplx messages
	mltplx string
	//Session is teh sessin name
	session string
	//path to teh scripts directry
	scripts string
	//git tells us if git version control is on
	git bool

	//id generator stuff
	id     chan string
	prefix string = "mrc_"
)

var (
	watcher, scriptWatch *fsnotify.Watcher
	err                  error
	msgout               chan interface{}
	server               string
	host, port           string
	ip                   net.IP
)

func init() {
	flag.StringVar(&watch, "watch", "./watch", "Specifies the path to watch, default to ./watch")
	//flag.StringVar(&netp, "netp", "tcp", "network protocol to use, defoults to TCP")
	flag.StringVar(&mltplx,
		"mltplx", "127.0.0.1:32123", "IP:PORT of multiplexer, defaults to localhost:32124")
	flag.StringVar(&session, "session", "", "Session name, should be something meaningful")
	//flag.StringVar(&scripts, "scripts",
	//	"./scripts", "Specifies the path to the scripts. Defaults to ./scripts. ")
	flag.StringVar(&server, "http", ":8089",
		"IP:PORT for the webserver serving files from the watch directory, default :8089.")

	initID()
}

func main() {
	flag.Parse()
	scripts = path.Join(watch, "scripts")
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	err := initSession()
	if err != nil {
		log.Fatal(err)
	}
	err = GitInit()
	if err == nil {
		git = true
	}
	msgout, err = NetSender()

	if err != nil {
		log.Fatal(err)
	}
	watcher, err = fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	//err = watcher.WatchFlags(watch, fsnotify.FSN_ALL)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//scriptWatch, err = fsnotify.NewWatcher()
	//if err != nil {
	//	log.Fatal(err)
	//}
	//err = scriptWatch.WatchFlags(scripts, fsnotify.FSN_MODIFY|fsnotify.FSN_CREATE)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//filepath stuff
	err = filepath.Walk(watch, WatchWalk)
	if err != nil {
		log.Println(err)
	}

	//random use of fsnotify
	go ListenWatcher(msgout)
	//go ScriptWatchListen()
	//start the http server
	//if server != "" {
	go func() { log.Fatal(http.ListenAndServe(server, http.FileServer(http.Dir(watch)))) }()
	///}

	host, port, err = net.SplitHostPort(server)
	if err != nil {
		log.Fatal(err)
	}
	if host == "" {
		ip, err = localIP()
		if err != nil {
			log.Fatal(err)
		}
		log.Println("ip: ", ip)
		host = ip.String()
	}
	log.Println("host ", host, "port ", port)
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT)
	<-ch
	log.Println("CTRL-C")
	// shutdown()
	cerr := watcher.Close()
	if err != nil {
		log.Println(cerr)
	}
	os.Exit(0)
}

//Intializes the session with all the parameters received from the command line.
//In particular it creates the missing paths and assign adefault sessino name.

func initSession() error {
	if session == "" {
		session = fmt.Sprintf("Mr.Clean session: %s", time.Now().Format(time.RFC3339))
		//session = "Mr.Clean session" //time.Now().Format(time.RFC3339)
	}
	//check if path to watch does not exist
	if _, err := os.Stat(watch); os.IsNotExist(err) {
		log.Printf("watch path: %s does not exixsts. Creaitng path.\n", watch)
		err = os.MkdirAll(watch, os.ModePerm)
		if err != nil {
			return err.(*os.PathError)
		}
	}
	//log.Println("scripts: ", scripts)
	//scripts, err := filepath.Abs(scripts)
	//if err != nil {
	//	return err
	//}
	log.Println("scripts: ", scripts)
	if _, err := os.Stat(scripts); os.IsNotExist(err) {
		log.Printf("scripts path: %s does not exixsts. Creaitng path.\n", scripts)
		err = os.MkdirAll(scripts, os.ModePerm)
		if err != nil {
			return err.(*os.PathError)
		}
	}
	return nil
}

//the filepath.WalkFunc to walk the watched folders
func WatchWalk(path string, info os.FileInfo, err error) error {
	//log.Println(path, info.IsDir())
	if err != nil {
		return err
	}
	if info.IsDir() {
		//err = watcher.WatchFlags(path, fsnotify.FSN_MODIFY|fsnotify.FSN_CREATE)
		err = watcher.Add(path)
		if err != nil {
			return err
		}
	}
	return nil
}

//check the size of a file
func fsize(fname string) int64 {
	info, err := os.Stat(fname)
	if err != nil {
		log.Println(err)
		return 0
	}
	return info.Size()
}

func finfo(fname string) (os.FileInfo, error) {
	info, err := os.Stat(fname)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return info, nil
}

//ListenWatcher listens to the watrcher and implments the logic.
//We register the creation  events in a set and when we receive
//a modify event (no attribute, a proper file write) we consider the
//file completed. Now we can safaly read the image.
//The events from the OS area  bit strange:
// trash: RENAME
// copy: CREATE, CHMOD, CHMOD, WRITE|CHMOD, CHMOD, CHMOD, CHMOD,CHMOD
// delete: REMOVE
// move: CREATE, CHMOD
// save vim: RENAME, CREATE, CHMOD
// append: WRITE, CHMOD
func ListenWatcher(out chan interface{}) {
	reImg := regexp.MustCompile(`\.(PNG|png|JPEG|JPG|jpeg|jpg)$`)
	reScr := regexp.MustCompile(`\.(r|R)$`)
	// this is the bit mask to filter image file events. We use
	//crete and write and teh file zize to see if the file is ready.
	bitmask := fsnotify.Write | fsnotify.Create
	var (
		info os.FileInfo
		err  error
	)
	for {
		select {
		case ev := <-watcher.Events:
			//get file info fo the event, we add new dirs to the watch
			info, err = finfo(ev.Name)
			if err != nil {
				log.Println(err)
				continue
			}
			if info.IsDir() {
				err = watcher.Add(ev.Name)
				if err != nil {
					log.Println(err)
					continue
				}
			}
			switch {
			default:
				continue
			case reImg.MatchString(filepath.Ext(ev.Name)):
				//got an image if it's a write or a create event we look for the
				//file size, a create with a 0 file size means the file is being copied
				// ans is not ready. We continue ans wait for a write event.
				//fsz := fsize(ev.Name)
				//log.Printf("fsize  of %s: %d\n", ev.Name, fsz)
				if ev.Op&bitmask != 0 && fsize(ev.Name) > 0 {
					img, err := NewImageData(ev.Name)
					if err != nil {
						log.Println("Error: ", err)
						continue
					}
					//send img data to MR. Clean
					out <- img
				} else {
					continue
				}
			case reScr.MatchString(filepath.Ext(ev.Name)):
				//anything ut a chmod is good for a script
				if ev.Op&fsnotify.Chmod == fsnotify.Chmod {
					continue
				}
			}
			if git {
				// git stuff
				cmd := exec.Command("git", "add", ev.Name)
				cmd.Dir = scripts
				err = cmd.Run()
				if err != nil { //no error means a git repo is ther already
					log.Printf("error adding %s to repo %v\n", ev.Name, err)
					log.Printf("args: %v\n", cmd.Args)
					continue
				}
				message := fmt.Sprintf("\"%s %s\"", session, time.Now().Format(time.RFC3339))
				cmd = exec.Command("git", "commit", "-am", message)
				cmd.Dir = scripts
				err = cmd.Run()
				if err != nil { //no error means a git repo is ther already
					log.Printf("error committing %s to repo %v\n", ev.Name, err)
					log.Printf("args: %v\n", cmd.Args)
					continue
				}
			}
		case err := <-watcher.Errors:
			log.Println("error:", err)
		}
	}
}

//GitInit() initialized a git repo in the watch folder. If a repo is present it fails
//greacefully and the version tracking wiht git is disabled. If the vwrsoin tracking is
//disabled. If the versoin tracking is disable Mr. Clean keep working.
func GitInit() error {
	//check if git is installed
	cmd := exec.Command("git", "--version")
	cmd.Dir = watch
	//cmd.Dir = scripts
	err := cmd.Run()
	if err != nil {
		log.Println("WARNING: git not found, script verson control disabled")
		return err
	}
	log.Println("git found")
	cmd = exec.Command("git", "status")
	cmd.Dir = watch
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err = cmd.Run()
	if err == nil { //no error means a git repo is ther already
		log.Printf("WARNING: a git repo is already in place in %s , script verson control disabled\n", scripts)
		return err
	}
	//log.Println("no error, scanning")
	scanner := bufio.NewScanner(&out)
	if !scanner.Scan() {
		log.Println("no output from git, script verson control disabled")
		log.Println(out.String())
		return err
	}
	firstLine := scanner.Bytes()
	if !bytes.HasPrefix(firstLine, []byte("fatal: Not a git repository")) {
		log.Println("unexpected result of 'git status': ", firstLine)
		log.Println("script verson control disabled")
		return err
	}
	log.Printf("no git repo found in %s, initializing one for Mr.Clean\n", scripts)
	cmd = exec.Command("git", "init")
	cmd.Dir = watch
	//cmd.Dir = scripts
	err = cmd.Run()
	if err != nil {
		log.Println("WARNING: git init faild, script verson control disabled")
		return err

	}
	return nil
}
func ScriptWatchListen() {

	//check if git is installed
	cmd := exec.Command("git", "--version")
	cmd.Dir = scripts
	err := cmd.Run()
	if err != nil {
		log.Println("WARNING: git not found, script verson control disabled")
		return
	}
	log.Println("git found")
	cmd = exec.Command("git", "status")
	cmd.Dir = scripts
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err = cmd.Run()
	if err == nil { //no error means a git repo is ther already
		log.Printf("WARNING: a git repo is already in place in %s , script verson control disabled\n", scripts)
		return
	}
	//log.Println("no error, scanning")
	scanner := bufio.NewScanner(&out)
	if !scanner.Scan() {
		log.Println("no output from git, script verson control disabled")
		log.Println(out.String())
		return
	}
	firstLine := scanner.Bytes()
	if !bytes.HasPrefix(firstLine, []byte("fatal: Not a git repository")) {
		log.Println("unexpected result of 'git status': ", firstLine)
		log.Println("script verson control disabled")
		return
	}
	log.Printf("no git repo found in %s, initializing one for Mr.Clean\n", scripts)
	cmd = exec.Command("git", "init")
	cmd.Dir = scripts
	err = cmd.Run()
	if err != nil {
		log.Println("WARNING: git init faild, script verson control disabled")
		return
	}
	for {
		select {
		case ev := <-scriptWatch.Events:
			if strings.HasSuffix(ev.Name, ".r") ||
				strings.HasSuffix(ev.Name, ".R") {
				//modified an R file, snapshot it in the git repo
				cmd = exec.Command("git", "add", ev.Name)
				cmd.Dir = scripts
				err = cmd.Run()
				if err != nil { //no error means a git repo is ther already
					log.Printf("error adding %s to repo %v\n", ev.Name, err)
					log.Printf("args: %v\n", cmd.Args)
					continue
				}
				message := fmt.Sprintf("\"%s %s\"", session, time.Now().Format(time.RFC3339))
				cmd = exec.Command("git", "commit", "-am", message)
				cmd.Dir = scripts
				err = cmd.Run()
				if err != nil { //no error means a git repo is ther already
					log.Printf("error committing %s to repo %v\n", ev.Name, err)
					log.Printf("args: %v\n", cmd.Args)
					continue
				}

			}
		case err := <-scriptWatch.Errors:
			log.Println("error: ", err)
		}
	}

}

//Get the image data form the event path and creta a new ImageData value.
//We strip the 'watch' var riable vaue from the path.
func NewImageData(path string) (*mrclean.ImageData, error) {
	relpath, err := filepath.Rel(watch, path)
	if err != nil {
		log.Fatal("Path to 'watch' mismatched: ", err)
	}
	dir, _ := filepath.Split(relpath)
	meta := strings.Split(dir, string(filepath.Separator))
	//we use the path form the fs event
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	cfg, _, err := image.DecodeConfig(file)
	if err != nil {
		log.Printf("Error %v opening image %s \n", err)
		return nil, err
	}
	size := [2]int{cfg.Width, cfg.Height}
	url := fmt.Sprintf("http://%s:%s/%s", host, port, relpath)

	ret := &mrclean.ImageData{
		URL: url,
		MetaData: mrclean.MetaData{
			Task:      meta[0],
			Approach:  meta[1],
			Iteration: meta[2],
			Method:    meta[3]},
		//Meta: meta,
		Name: <-id,
		Size: size,
	}

	return ret, nil
}

type ImageData struct {
	Path      string
	Task      string
	Approach  string
	Iteration string //int
	Method    string
}

func NetSender() (chan interface{}, error) {
	//conn, err := net.ListenMulticastUDP("udp", nil, message.Mcast)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//err = conn.SetWriteBuffer(1 << 12) //4Kb
	//if err != nil {
	//	log.Fatal(err)
	//}

	bcastconn, err := net.DialUDP("udp", nil, mrclean.LB)
	//conn, err := net.Dial("udp", message.Mcast.String())
	//bcastconn, err := message.JoinMcast(message.Mcast.IP)
	if err != nil {
		log.Println("Error joinig multicast ", err)
	}
	//log.Println("conn", conn)
	//bcastconn, ok := conn.(*net.UDPConn)
	//if !ok {
	//	log.Println("bcastconn not UDP")
	//}
	//big buffer out
	if err := bcastconn.SetWriteBuffer(1 << 21); err != nil {
		log.Println("error setting write buffer, ", err)
		return nil, err
	}
	//wall, err := ecs.DialSenderJF(net, addr)
	//conn, err := net.Dial(network, addr)
	//if err != nil {
	//	log.Printf("Error Dialing: %+v\n", err)
	//	return nil, err
	//}
	log.Println("encoding to ", bcastconn.RemoteAddr())
	//enc := json.NewEncoder(bcastconn)
	ch := make(chan interface{}, 5) //buffer some just in case
	go func() {
		for i := range ch {
			//sending a message containig an image
			msg := mrclean.OutMessage{
				Header:  mrclean.ImageMsg,
				Content: i,
			}
			buff, err := json.Marshal(msg)
			if err != nil {
				log.Fatal(err)
			}
			log.Println("Sending ", string(buff))
			//msg := &ecs.InputMessage{}
			//err = json.Unmarshal(buff, msg)

			//if err != nil {
			//	log.Println()

			//}
			//log.Printf("decoding %v\n", msg)

			//err = enc.Encode(msg)
			_, err = bcastconn.Write(buff)
			if err != nil {
				log.Println(err)
			}
		}
	}()
	return ch, nil
}

func initID() chan string {
	id = make(chan string)
	count := 0
	go func() {
		for {
			select {
			case id <- prefix + strconv.Itoa(count):
				count++
			}
		}
	}()
	return id

}

func localIP() (net.IP, error) {
	tt, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, t := range tt {
		aa, err := t.Addrs()
		if err != nil {
			return nil, err
		}
		for _, a := range aa {
			ipnet, ok := a.(*net.IPNet)
			if !ok {
				continue
			}
			v4 := ipnet.IP.To4()
			if v4 == nil || v4[0] == 127 { // loopback address
				continue
			}
			return v4, nil
		}
	}
	return nil, fmt.Errorf("cannot find local IP address")
}
