package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
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

var coord parseStruct

func main() {
	pidFlag := flag.Int("p", 0, "pid for check file descriptors count")
	allFDFlag := flag.Bool("a", false, "check system file descriptors count")
	listFDFlag := flag.Bool("l", false, "list file descriptors count for all process")

	flag.Parse()

	if pidFlag != nil && *pidFlag != 0 {
		fmt.Printf("Opened file descriptors for pid %d: %d\n", *pidFlag, countByDirectory(*pidFlag))
	} else if allFDFlag != nil && *allFDFlag {
		// log.Println(countAllPids())
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

func getListPIDFD() {
	out, err := exec.Command("/bin/sh", "-c", "lsof -n").Output()
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	var parsedSlice []string
	pidOfCount := make(map[string]map[string]processStruct)
	rowsSlice := strings.Split(string(out), "\n")
	for i := range rowsSlice {
		if i == 0 {
			coord = getCollCoordinates(rowsSlice[i])
			continue
		}
		parsedSlice = parseRowString(rowsSlice[i])
		if parsedSlice == nil {
			continue
		}
		if strings.TrimSpace(parsedSlice[2]) == "" {
			parsedSlice[2] = "unknown"
		}
		if tmp, ok := pidOfCount[parsedSlice[1]]; ok {
			if tmp1, ok := tmp[parsedSlice[2]]; ok {
				tmp1.Count++
				tmp[parsedSlice[2]] = tmp1
			} else {
				pidOfCount[parsedSlice[1]][parsedSlice[2]] = processStruct{
					Command: parsedSlice[0],
					Count:   1,
				}
			}
			pidOfCount[parsedSlice[1]] = tmp

		} else {
			tmp := make(map[string]processStruct)
			tmp[parsedSlice[2]] = processStruct{
				Command: parsedSlice[0],
				Count:   1,
			}
			pidOfCount[parsedSlice[1]] = tmp
		}
	}
	tableData := make([][]string, 0)
	for key := range pidOfCount {
		for key2 := range pidOfCount[key] {
			tableData = append(tableData, []string{pidOfCount[key][key2].Command, key, key2, strconv.Itoa(pidOfCount[key][key2].Count)})
		}
	}
	sort.Slice(tableData, func(i, j int) bool {
		iint, _ := strconv.Atoi(tableData[i][3])
		jint, _ := strconv.Atoi(tableData[j][3])

		return iint > jint
	})
	fmt.Println("Count of opened file descriptors by pid:")
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"COMMAND", "PID", "FDTYPE", "COUNT"})
	table.AppendBulk(tableData)
	table.Render()
}

func parseRowString(row string) []string {
	if len(row) > coord.COMMANDEnd {
		collSlice := make([]string, 3)

		collSlice[0] = strings.TrimSpace(row[:coord.COMMANDEnd])
		collSlice[1] = strings.TrimSpace(row[coord.PIDStart:coord.PIDEnd])
		collSlice[2] = strings.TrimSpace(row[coord.TYPEStart:coord.TYPEEnd])
		return collSlice
	}
	return nil
}

func getCollCoordinates(row string) parseStruct {
	coordinates := parseStruct{
		COMMANDStart: 0,
	}
	for i := strings.Index(row, "COMMAND"); row[i] != 32; i++ {
		coordinates.COMMANDEnd = i + 3
	}
	for i := strings.Index(row, "PID"); row[i] != 32; i++ {
		coordinates.PIDEnd = i + 1
	}
	for i := strings.Index(row, "TID"); row[i] != 32; i++ {
		coordinates.TIDEnd = i + 1
	}
	for i := strings.Index(row, "TASKCMD"); row[i] != 32; i++ {
		coordinates.TASKCMDEnd = i + 3
	}
	for i := strings.Index(row, "USER"); row[i] != 32; i++ {
		coordinates.USEREnd = i + 3
	}
	for i := strings.Index(row, "FD"); row[i] != 32; i++ {
		coordinates.FDEnd = i + 3
	}
	for i := strings.Index(row, "TYPE"); row[i] != 32; i++ {
		coordinates.TYPEEnd = i + 1
	}
	for i := strings.Index(row, "DEVICE"); row[i] != 32; i++ {
		coordinates.DEVICEEnd = i + 3
	}
	for i := strings.Index(row, "SIZE/OFF"); row[i] != 32; i++ {
		coordinates.SIZEOFFEnd = i + 3
	}
	for i := strings.Index(row, "NODE"); row[i] != 32; i++ {
		coordinates.NODEEnd = i + 3
	}
	coordinates.PIDStart = coordinates.COMMANDEnd
	coordinates.TIDStart = coordinates.PIDEnd
	coordinates.TASKCMDStart = coordinates.TIDEnd
	coordinates.USERStart = coordinates.TASKCMDEnd
	coordinates.FDStart = coordinates.USEREnd
	coordinates.TYPEStart = coordinates.FDEnd
	coordinates.DEVICEStart = coordinates.TYPEEnd
	coordinates.SIZEOFFStart = coordinates.DEVICEEnd
	coordinates.NODEStart = coordinates.SIZEOFFEnd
	coordinates.NAMEStart = strings.Index(row, "NAME")
	return coordinates
}
