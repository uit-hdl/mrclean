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

	//display
	display := exec.Command("display")
	//get stdout
	stdoutd, err := display.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	//go io.Copy(os.Stdout, stdoutd)

	//core
	core := exec.Command("core")
	stdoutc, err := core.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	//go io.Copy(os.Stdout, stdoutc)

	//chronicle
	chronicle := exec.Command("chronicle")
	stdoutcr, err := chronicle.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	//go io.Copy(os.Stdout, stdoutcr)

	//Write combined stdout of the comands
	go io.Copy(os.Stdout, io.MultiReader(stdoutd, stdoutc, stdoutcr))

	//map containing the commands and their pth as keys
	cmdm := make(map[string]*exec.Cmd)

	//start the commands and puththem in the map
	err = display.Start()
	if err != nil {
		log.Fatalf("Start error %+v\n", err)
	}
	cmdm[display.Path] = display
	//this is gross and wrong but it needs some time to connect...
	time.Sleep(1 * time.Second)

	err = core.Start()
	if err != nil {
		display.Process.Kill()
		log.Fatalf("%+v\n", err)
	}
	cmdm[core.Path] = core
	//this is gross and wrong but it needs some time to connect...
	time.Sleep(1 * time.Second)

	err = chronicle.Start()
	if err != nil {
		core.Process.Kill()
		display.Process.Kill()
		log.Fatalf("%+v\n", err)
	}
	cmdm[chronicle.Path] = chronicle
	//this is gross and wrong but it needs some time to connect...
	time.Sleep(1 * time.Second)

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
