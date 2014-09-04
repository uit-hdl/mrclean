/* Chronicle is a aomponent that monitors a directory in the file sysems
and provide a chronicle of the changes in it.




*/
package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"net/rpc/jsonrpc"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"syscall"
	"time"

	"github.com/UniversityofTromso/mrclean"
	"github.com/fsnotify/fsnotify"
)

// Chronicle is the type recording the changes in the file system. Its method
// have the signature to be exported via RPC.
type Chronicle struct {
}

var (
	//IP:PORT of thr rpc server
	rpcserver string
	//session name, will apper in each commit
	session string
	//path to the base directory to watch for the chronicle
	watch string
	//path to scripts relatice to the watch path
	scripts string = "scripts"
	//ip:port of the image URL
	ip, port string = "127.0.0.1", "8089"
	//Core component RPC server address
	corerpc string
	//netconn is the transport protocol for the connection
	netconn string

	client  *rpc.Client
	watcher *fsnotify.Watcher
	git     bool
	err     error
)

func init() {
	flag.StringVar(&session,
		"session", "mrclean", "Name of the session.")
	flag.StringVar(&corerpc,
		"corerpc", mrclean.CoreAddr,
		"IP:PORT of the Core RPC server")
	flag.StringVar(&rpcserver,
		"rpcserver", mrclean.ChronicleAddr,
		"IP:PORT of the rpc server, defaults to localhost:32124")
	flag.StringVar(&watch, "watch", "./watch",
		"Specifies the path to watch, default to ./watch")
	flag.StringVar(&netconn, "net", "tcp", "Specifies the connection protocol: tcp, udp, unix etc..")
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func main() {
	flag.Parse()
	client, err = jsonrpc.Dial(netconn, corerpc)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	err := initSession()
	if err != nil {
		log.Fatal(err)
	}
	//var reply int
	//args := mrclean.Visual{Name: "scatterplot.jpg"}
	//err = client.Call("Core.AddVisual", args, &reply)
	//if err != nil {
	//	log.Fatal("arith error:", err)
	//}
	//log.Println("Got reply from Core: ", reply)
	err = GitInit()
	if err == nil {
		git = true
	}
	watcher, err = fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	//get local ip for URL in the Visuals
	nip, err := localIP()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("ip: ", nip)
	ip = nip.String()
	//walk the file path for watching
	err = filepath.Walk(watch, WatchWalk)
	if err != nil {
		log.Println(err)
	}
	//actually qwatch teh path
	go ListenWatcher(client)
	//start the http server
	//is messy i know
	go func() {
		log.Fatal( // if somethng breaks we stop and log
			http.ListenAndServe(ip+":"+port,
				//http.StripPrefix( watch,
				http.FileServer(http.Dir(watch)))) //)
	}()

	//wait for CTRL-C
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT)
	<-ch
	log.Println("CTRL-C")
	// shutdown()
	cerr := watcher.Close()
	if err != nil {
		log.Println(cerr)
	}

}

func GitInit() error {
	//print working dir
	cmd := exec.Command("pwd")
	//cmd.Dir = watch
	//cmd.Dir = scripts
	err = cmd.Run()
	if err != nil {
		log.Println("pwd error")
		return err

	}
	//check if git is installed
	cmd = exec.Command("git", "--version")
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
		log.Printf("A git repo is already in place\n")
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
	//cmd.Dir = watch
	//cmd.Dir = scripts
	err = cmd.Run()
	if err != nil {
		log.Println("WARNING: git init faild, script verson control disabled")
		return err

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

//ListenWatcher listens to the watcher and implments the logic.
//We register the creation  events in a set and when we receive
//a modify event (no attribute, a proper file write) we consider the
//file completed. Now we can safaly read the image.
//The events from the OS area  bit strange:
// trash: RENAME
// copy: CREATE, CHMOD, CHMOD, WRITE|CHMOD, CHMOD, CHMOD, CHMOD, CHMOD
// delete: REMOVE
// move: CREATE, CHMOD
// save vim: RENAME, CREATE, CHMOD
// append: WRITE, CHMOD
func ListenWatcher(core *rpc.Client) {
	reImg := regexp.MustCompile(`\.(PNG|png|JPEG|JPG|jpeg|jpg)$`)
	reScr := regexp.MustCompile(`\.(r|R)$`)
	hidden := regexp.MustCompile(`^\. | /\.`)
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
			if hidden.MatchString(ev.Name) { //hidden file in path
				//log.Println("HIDDEN", ev.Name)
				continue
			}

			////////////////////////////
			//relpath, err := filepath.Rel(watch, ev.Name)
			//if err != nil {
			//	log.Fatal("Path to 'watch' mismatched: ", err)
			//}
			//log.Println("path", ev.Name, "relative", relpath)
			/////////////////////////////////
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
			relpath, err := filepath.Rel(watch, ev.Name)
			if err != nil {
				log.Fatal("Path to 'watch' mismatched: ", err)
			}
			switch {
			default:
				continue
			case reImg.MatchString(filepath.Ext(ev.Name)):
				//got an image if it's a write or a create event we look for the
				//file size, a create with a 0 file size means the file is being copied
				// and is not ready. We continue and wait for a write event.
				//fsz := fsize(ev.Name)
				//log.Printf("fsize  of %s: %d\n", ev.Name, fsz)
				if ev.Op&bitmask != 0 && fsize(ev.Name) > 0 {
					commit, err := gitCommit(ev.Name)
					if err != nil {
						log.Println(err)
						continue
					}

					arg := mrclean.Visual{
						Name:   relpath,
						ID:     0,
						URL:    fmt.Sprintf("http://%s:%s/%s", ip, port, relpath),
						Commit: commit,
						//Origin: []float64{0,0},
						//Size: []float64{0,0},

					}
					arg.Rectangle.Max.X, arg.Rectangle.Max.Y, err = imgsize(ev.Name)
					if err != nil {
						log.Println(err)
					}
					log.Printf("Adding %+v\n", arg)
					var reply *int
					//send img data to MR. Clean
					err = client.Call("Core.AddVisual", arg, &reply)
					if err != nil {
						log.Println("Core.AddVisual error:", err)
					}
					if reply != nil && *reply != 0 {
						log.Println("Error adding Visual!!")
					}
				} else {
					continue
				}
			case reScr.MatchString(filepath.Ext(ev.Name)):
				//anything but a chmod is good for a script
				if ev.Op&fsnotify.Chmod == fsnotify.Chmod {
					continue
				}
				//commit here the change in the script
				commit, err := gitCommit(ev.Name)
				if err != nil {
					log.Println("Error comitting code ", err)
					continue
				}
				log.Println("Committed change in code, commit: ", commit)

			} //switch
			//if git {
			//	// git stuff
			//	cmd := exec.Command("git", "add", ev.Name)
			//	cmd.Dir = watch
			//	err = cmd.Run()
			//	if err != nil { //no error means a git repo is ther already
			//		log.Printf("error adding %s to repo %v\n", ev.Name, err)
			//		log.Printf("command: %+v\n", cmd)
			//		//log.Printf("args: %v\n", cmd.Args)
			//		continue
			//	}
			//	message := fmt.Sprintf("\"%s %s\"", session, time.Now().Format(time.RFC3339))
			//	cmd = exec.Command("git", "commit", "-am", message)
			//	cmd.Dir = watch
			//	err = cmd.Run()
			//	if err != nil { //no error means a git repo is ther already
			//		log.Printf("error committing %s to repo %v\n", ev.Name, err)
			//		log.Printf("command: %+v\n", cmd)
			//		continue
			//	}
			//}
		case err := <-watcher.Errors:
			log.Println("error:", err)
		}
	}
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

func finfo(fname string) (os.FileInfo, error) {
	info, err := os.Stat(fname)
	if err != nil {
		//log.Println(err)
		return nil, err
	}
	return info, nil
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

func imgsize(path string) (int, int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()
	cfg, _, err := image.DecodeConfig(file)
	if err != nil {
		log.Printf("Error %v opening image %s \n", err, path)
		return 0, 0, err
	}
	return cfg.Width, cfg.Height, nil
}

func gitCommit(name string) (string, error) {
	// git stuff
	cmd := exec.Command("git", "add", name)
	//cmd.Dir = watch
	err = cmd.Run()
	if err != nil { //no error means a git repo is ther already
		log.Printf("error adding %s to repo %v\n", name, err)
		log.Printf("command: %+v\n", cmd)
		//log.Printf("args: %v\n", cmd.Args)
		return "", err
	}
	message := fmt.Sprintf("\"%s %s\"", session, time.Now().Format(time.RFC3339))
	cmd = exec.Command("git", "commit", "-am", message)
	cmd.Dir = watch
	out, err := cmd.Output()
	if err != nil { //no error means a git repo is ther already
		log.Printf("error committing %s to repo %v\n", name, err)
		log.Printf("command: %+v\n", cmd)
		return "", err
	}
	lines := bytes.Split(out, []byte{'\n'})
	first := lines[0]
	//fmt.Printf("first:\n%s\n", first)
	words := bytes.Split(first, []byte{' '})
	var ret string
	for _, w := range words {
		if w[len(w)-1] == ']' {
			//fmt.Printf("Commit #: %s\n", w[:len(w)-1])
			ret = string(w[:len(w)-1])
			break
		}
	}
	return ret, nil
}
