package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/olekukonko/tablewriter"
)

type parseStruct struct {
	COMMANDStart, PIDStart, TIDStart,
	TASKCMDStart, USERStart, FDStart,
	TYPEStart, DEVICEStart, SIZEOFFStart,
	NODEStart, NAMEStart, COMMANDEnd,
	PIDEnd, TIDEnd, TASKCMDEnd,
	USEREnd, FDEnd, TYPEEnd,
	DEVICEEnd, SIZEOFFEnd, NODEEnd int
}
type processStruct struct {
	Command string
	Count   int
}

const macOS = "darwin"

var coord parseStruct

func main() {
	pidFlag := flag.Int("p", 0, "pid for check file descriptors count")
	allFDFlag := flag.Bool("a", false, "check system file descriptors count")
	listFDFlag := flag.Bool("l", false, "list file descriptors count for all process")

	flag.Parse()
	if pidFlag != nil && *pidFlag != 0 {
		if runtime.GOOS == macOS {
			fmt.Printf("Opened file descriptors for pid %d: %d\n", *pidFlag, countPIDsOpenFiles(*pidFlag))
		} else {
			fmt.Printf("Opened file descriptors for pid %d: %d\n", *pidFlag, countByDirectory(*pidFlag))
		}
	} else if allFDFlag != nil && *allFDFlag {
		fmt.Printf("Opened file descriptors: %d\n", countOpenFiles())
	} else if *listFDFlag {

		getListPIDFD()
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

func countPIDsOpenFiles(pid int) int64 {
	out, err := exec.Command("/bin/sh", "-c", fmt.Sprintf("lsof -p %d", pid)).Output()
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

type commandStruct struct {
	command string
	pid     string
	data    []fdStruct
}
type fdStruct struct {
	fd    string
	ftype string
}

func getListPIDFD() {
	out, err := exec.Command("/bin/sh", "-c", "lsof -F ct").Output()
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	rowsSlice := strings.Split(string(out), "\n")
	commandMap := make(map[string]commandStruct)
	currentPID := ""
	currentCommandStruct := fdStruct{}
	for i := range rowsSlice {
		fieldType, val := parseFRowString(rowsSlice[i])
		switch fieldType {
		case 0:
			commandMap[val] = commandStruct{pid: val}
			currentPID = val
		case 1:
			if mval, ok := commandMap[currentPID]; ok {
				mval.command = val
				commandMap[currentPID] = mval
			}
		case 2:
			currentCommandStruct = fdStruct{fd: val}
		case 3:
			currentCommandStruct.ftype = val
			if mval, ok := commandMap[currentPID]; ok {
				mval.data = append(mval.data, currentCommandStruct)
				commandMap[currentPID] = mval
			}
		case -1:
			continue
		}
	}
	outData := make([][]string, 0)
	for _, v := range commandMap {
		fTypeCount := make(map[string]int)
		for _, vd := range v.data {
			fTypeCount[vd.ftype]++
		}
		for key, ft := range fTypeCount {
			outData = append(outData, []string{v.command, v.pid, key, strconv.Itoa(ft)})
		}
	}
	sort.Slice(outData, func(i, j int) bool {
		iint, err := strconv.Atoi(outData[i][3])
		if err != nil {
			log.Fatal(fmt.Sprintf("getListPIDFD convert string to int error: %s", err))
		}
		jint, err := strconv.Atoi(outData[j][3])
		if err != nil {
			log.Fatal(fmt.Sprintf("getListPIDFD convert string to int error#2: %s", err))
		}
		return iint > jint
	})
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"COMMAND", "PID", "FDTYPE", "COUNT"})
	table.AppendBulk(outData)
	table.Render()
}

func parseFRowString(row string) (int8, string) {
	if len(row) == 0 {
		return -1, ""
	}
	switch string(row[0]) {
	case "c":
		return 1, row[1:]
	case "p":
		return 0, row[1:]
	case "f":
		return 2, row[1:]
	case "t":
		return 3, row[1:]
	default:
		return -1, ""
	}
}
