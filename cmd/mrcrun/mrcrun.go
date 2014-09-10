package main

import (
	"io"
	"log"
	"os"
	"os/exec"
	"time"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	display := exec.Command("display")
	//get stdout
	stdoutd, err := display.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}

	core := exec.Command("core")
	stdoutc, err := core.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}

	chronicle := exec.Command("chronicle")
	stdoutcr, err := chronicle.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}

	//map containing the commands and their pth as keys
	cmdm := make(map[string]*exec.Cmd)
	//start the commands and puththem in the map
	err = display.Start()
	if err != nil {
		log.Fatalf("Start error %+v\n", err)
	}
	//this is gross but it needs some tiem to connect...
	time.Sleep(1 * time.Second)
	//copy stdout of child process to stdout
	go io.Copy(os.Stdout, stdoutd)
	cmdm[display.Path] = display
	err = core.Start()
	if err != nil {
		display.Process.Kill()
		log.Fatalf("%+v\n", err)
	}
	go io.Copy(os.Stdout, stdoutc)
	cmdm[core.Path] = core
	err = chronicle.Start()
	if err != nil {
		core.Process.Kill()
		display.Process.Kill()
		log.Fatalf("%+v\n", err)
	}
	go io.Copy(os.Stdout, stdoutcr)

	cmdm[chronicle.Path] = chronicle
	//all running we wait for deaths
	//upon a death we remove the dead command fomr the map
	//and kill the others to clean up
	deadchan := make(chan string)
	wait2(display, deadchan)
	wait2(core, deadchan)
	wait2(chronicle, deadchan)
	dead := <-deadchan
	delete(cmdm, dead)
	for _, c := range cmdm {
		c.Process.Kill()
	}

}

func wait2(cmd *exec.Cmd, ch chan string) {
	go func() {
		err := cmd.Wait()
		if err != nil {
			log.Printf("Process %s ended with  error %v\n", cmd.Path, err)
		}
		ch <- cmd.Path
	}()
}
func wait(cmd exec.Cmd) chan struct{} {
	ch := make(chan struct{})
	go func() {
		err := cmd.Wait()
		if err != nil {
			log.Println(err)
		}
		ch <- struct{}{}
	}()
	return ch
}
