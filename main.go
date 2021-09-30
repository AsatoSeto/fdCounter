package main

import (
	"flag"
	"fmt"
	"os/exec"
	"strings"
)

func main() {
	pidFlag := flag.Int("p", 0, "pid for check file descriptors count")
	allFDFlag := flag.Bool("a", false, "check system file descriptors count")
	flag.Parse()

	if pidFlag != nil && *pidFlag != 0 {
		fmt.Printf("Opened file descriptors for pid %d: %d\n", *pidFlag, countOpenFilesForPid(*pidFlag))
	} else if allFDFlag != nil && *allFDFlag {
		fmt.Printf("Opened file descriptors: %d\n", countOpenFiles())
	} else {
		fmt.Printf("Invalid parameter. Use the ``-h'' option to get more help information.\n")
	}
}

func countOpenFilesForPid(pid int) int64 {
	out, err := exec.Command("/bin/sh", "-c", fmt.Sprintf("lsof -n -p %v", pid)).Output()
	if err != nil {
		fmt.Println(err.Error())
	}
	return int64(len(strings.Split(string(out), "\n")) - 1)

}

func countOpenFiles() int64 {
	out, err := exec.Command("/bin/sh", "-c", "lsof -n").Output()
	if err != nil {
		fmt.Println(err.Error())
	}
	return int64(len(strings.Split(string(out), "\n")) - 1)
}
