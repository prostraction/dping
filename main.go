package main

import (
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

// Queue collects ping metrics for defined duration
type Queue struct {
	items []map[int]int
}

// Push adds current one second's ping metric to Queue
func (q *Queue) Push(item map[int]int) {
	q.items = append(q.items, item)
}

// Pop removes last n seconds (hours) ago ping metric from Queue
func (q *Queue) Pop() map[int]int {
	if len(q.items) == 0 {
		return nil
	}
	item := q.items[0]
	q.items = q.items[1:len(q.items)]
	return item
}

var ipAdr = ""

var secondsPassed int
var queueMin Queue
var queueHour Queue
var queue3Hour Queue
var statsSecond map[int]int
var statsMin map[int]int
var statsHour map[int]int
var stats3Hour map[int]int
var statsAll map[int]int

// var seq int
var mu sync.Mutex

func log() {
	tMinCheck := time.Now()
	tLast := time.Now()
	for {
		tNow := time.Now()
		if tNow.Second()-tLast.Second() != 0 {
			mu.Lock()
			secondsPassed++

			queueMin.Push(statsSecond)
			queueHour.Push(statsSecond)
			queue3Hour.Push(statsSecond)

			if secondsPassed >= 60 {
				remOldSec := queueMin.Pop()
				for k, v := range remOldSec {
					if statsMin[k] > 0 {
						statsMin[k] -= v
					}
				}
			}
			if secondsPassed >= 3600 {
				remOldSec := queueHour.Pop()
				for k, v := range remOldSec {
					if statsHour[k] > 0 {
						statsHour[k] -= v
					}
				}
			}
			if secondsPassed >= 3*3600 {
				remOldSec := queue3Hour.Pop()
				for k, v := range remOldSec {
					if stats3Hour[k] > 0 {
						stats3Hour[k] -= v
					}
				}
			}

			statsSecond = nil
			mu.Unlock()
			tLast = tNow
		}
		if tNow.Minute()-tMinCheck.Minute() != 0 {
			mu.Lock()
			strTime := tNow.Format(time.Stamp)
			allPacketsMin := 0
			allPacketsHour := 0
			allPackets3Hour := 0
			allPacketsAll := 0

			droppedPacketsMin := 0
			droppedPacketsHour := 0
			droppedPackets3Hour := 0
			droppedPacketsAll := 0

			for k, v := range statsMin {
				if k >= 300 {
					droppedPacketsMin += v
				}
				allPacketsMin += v
			}
			for k, v := range statsHour {
				if k >= 300 {
					droppedPacketsHour += v
				}
				allPacketsHour += v
			}
			for k, v := range stats3Hour {
				if k >= 300 {
					droppedPackets3Hour += v
				}
				allPackets3Hour += v
			}
			for k, v := range statsAll {
				if k >= 300 {
					droppedPacketsAll += v
				}
				allPacketsAll += v
			}
			if allPacketsMin > 0 {
				fmt.Printf("[%s]    loss M: [%.2f%% (%d of %d)], H: [%.2f%% (%d of %d)], 3H: [%.2f%% (%d of %d)], ALL: [%.2f%% (%d of %d)]\n", strTime,
					100*float64(droppedPacketsMin)/float64(allPacketsMin),
					droppedPacketsMin, allPacketsMin,

					100*float64(droppedPacketsHour)/float64(allPacketsHour),
					droppedPacketsHour, allPacketsHour,

					100*float64(droppedPackets3Hour)/float64(allPackets3Hour),
					droppedPackets3Hour, allPackets3Hour,

					100*float64(droppedPacketsAll)/float64(allPacketsAll),
					droppedPacketsAll, allPacketsAll)
			}
			mu.Unlock()
			tMinCheck = tNow
		}
	}
}

func test() error {
	connWrite, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		return err
	}

	connWrite.SetDeadline(time.Now().Add(time.Millisecond * 150))
	ipAddr, err := net.ResolveIPAddr("ip4", ipAdr)
	if err != nil {
		return err
	}

	msg := icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.Echo{
			ID:   os.Getpid() & 0xffff,
			Seq:  1,
			Data: []byte("Hello, World!"),
		},
	}
	msgBytes, err := msg.Marshal(nil)
	if err != nil {
		return err
	}

	if _, err := connWrite.WriteTo(msgBytes, ipAddr); err != nil {
		return err
	}

	tBegin := time.Now()

	buf := make([]byte, 1500)
	n, _, _ := connWrite.ReadFrom(buf)
	if n < 0 {
		fmt.Println(n)
	}
	tLast := time.Now()
	mu.Lock()
	if statsSecond == nil {
		statsSecond = make(map[int]int)
	}
	latency := tLast.UnixMilli() - tBegin.UnixMilli()
	statsSecond[int(latency)]++
	statsHour[int(latency)]++
	stats3Hour[int(latency)]++
	statsMin[int(latency)]++
	statsAll[int(latency)]++
	mu.Unlock()
	connWrite.Close()
	return nil
}

func printHelp() {
	fmt.Println("USAGE: det-ping IPv4 [arguments]")
	//fmt.Println("IP address must be writen as IPv4 like 1.1.1.1 or 127.0.0.1.")
	fmt.Println("Available arguments: ")
	fmt.Println("-t [msec] or --timeout [msec]. (default -t 300)")
}

func main() {

	argsGiven := os.Args[1:]
	if len(argsGiven) < 1 {
		printHelp()
		return
	}
	if strings.Count(argsGiven[0], ".") != 3 {
		printHelp()
		return
	} else {
		ipAdr = argsGiven[0]
	}
	for i := 1; i < len(argsGiven); i += 2 {
		if i+1 < len(argsGiven) {
			fmt.Println(argsGiven[i], argsGiven[i+1])
			switch argsGiven[i] {
			case "-t":
				fallthrough
			case "--timeout":
				fmt.Println("TIMEOUT: ", argsGiven[i+1])
			default:
				fmt.Println("Unrecongnized command:", argsGiven[i])
				printHelp()
				return
			}
		} else {
			fmt.Println(argsGiven[i], "requires an argument.")
			printHelp()
			return
		}
	}

	queueMin = Queue{}
	queueHour = Queue{}
	queue3Hour = Queue{}

	secondsPassed = 0
	statsSecond = make(map[int]int)
	statsHour = make(map[int]int)
	stats3Hour = make(map[int]int)
	statsMin = make(map[int]int)
	statsAll = make(map[int]int)
	go log()
	for {
		err := test()
		if err != nil {
			fmt.Println(err.Error())
			break
		}
	}
}
