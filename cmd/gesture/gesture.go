package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"

	"log"

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
	//netconn is the transport protocol for the connection
	netconn string
)

func init() {
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
