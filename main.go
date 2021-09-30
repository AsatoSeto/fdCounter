package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
)

func main() {
	pidFlag := flag.Int("p", 0, "pid for check file descriptors count")
	allFDFlag := flag.Bool("a", false, "check system file descriptors count")
	flag.Parse()

	if pidFlag != nil && *pidFlag != 0 {
		fmt.Printf("Opened file descriptors for pid %d: %d\n", *pidFlag, countByDirectory(*pidFlag))
	} else if allFDFlag != nil && *allFDFlag {
		// log.Println(countAllPids())
		fmt.Printf("Opened file descriptors: %d\n", countOpenFiles())
	} else {
		fmt.Printf("Invalid parameter. Use the ``-h'' option to get more help information.\n")
	}
}

func countByDirectory(pid int) int64 {
	dirEntr, err := os.ReadDir(fmt.Sprintf("/proc/%d/fd", pid))
	if err != nil {
		fmt.Println(err.Error())
		return 0
	}
	return int64(len(dirEntr))
}

func countOpenFiles() int64 {
	out, err := exec.Command("/bin/sh", "-c", "lsof -n").Output()
	if err != nil {
		fmt.Println(err.Error())
		return 0
	}
	return int64(len(strings.Split(string(out), "\n")) - 1)
}

func countAllPids() int64 {
	var mu sync.Mutex
	pids := getPids()
	count := 0
	wg := sync.WaitGroup{}
	for i := range pids {
		wg.Add(1)
		go func(pid string, wgr *sync.WaitGroup) {
			if len(pid) == 0 {
				wg.Done()
				return
			}
			dirEntr, err := os.ReadDir(fmt.Sprintf("/proc/%s/fd", strings.TrimSpace(pid)))
			if err != nil {
				if !os.IsNotExist(err) {
					fmt.Println(err.Error())
					wg.Done()
					return
				}
			}
			mu.Lock()
			count += len(dirEntr)
			mu.Unlock()
			wg.Done()
		}(pids[i], &wg)
	}
	wg.Wait()
	return int64(count)
}

func getPids() []string {
	out, err := exec.Command("/bin/sh", "-c", "ps axo pid").Output()
	if err != nil {
		fmt.Println("err", err.Error())
		return nil
	}
	return strings.Split(string(out), "\n")[1:]
}
