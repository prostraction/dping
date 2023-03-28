package main

import (
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

// Contains collected pings
type Queue struct {
	items []map[int]int
}

func (q *Queue) Push(item map[int]int) {
	q.items = append(q.items, item)
}
func (q *Queue) Pop() map[int]int {
	if len(q.items) == 0 {
		return nil
	}
	item := q.items[0]
	q.items = q.items[1:len(q.items)]
	return item
}

var ipAdr = "127.0.0.1"

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

func test() {
	connWrite, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		fmt.Println(err)
		//log.Fatal(err)
	}

	connWrite.SetDeadline(time.Now().Add(time.Millisecond * 150))
	ipAddr, err := net.ResolveIPAddr("ip4", ipAdr)
	if err != nil {
		fmt.Println(err)
		//log.Fatal(err)
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
		fmt.Println(err)
		//log.Fatal(err)
	}

	if _, err := connWrite.WriteTo(msgBytes, ipAddr); err != nil {
		fmt.Println(err)
		//log.Fatal(err)
	}

	tBegin := time.Now()

	buf := make([]byte, 1500)
	n, _, _ := connWrite.ReadFrom(buf)
	if n < 0 {
		fmt.Println(n)
	}
	/*if err != nil {
		log.Fatal(err)
	}
	if peer.String() != ipAddr.String() {
		log.Fatalf("got reply from unexpected IP %s; want %s", peer, ipAddr)
	}*/

	/*
		replyMsg, err := icmp.ParseMessage(ipv4.ICMPTypeEchoReply.Protocol(), buf[:n])
		if err != nil {
			fmt.Println(err)
		}
		if replyMsg.Type != ipv4.ICMPTypeEchoReply {
			fmt.Println(err)
			//log.Fatalf("got %v; want %v", replyMsg.Type, ipv4.ICMPTypeEchoReply)
		}
	*/
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
	//if stats_min == nil {
	//	stats_min = make(map[int64]int64)
	//}
	//stats_min[t_last.UnixMilli()-t_begin.UnixMilli()]++
	//stats_all[t_last.UnixMilli()-t_begin.UnixMilli()]++
	mu.Unlock()
	//fmt.Printf("Got ICMP packet: %+v (%d ms.)\n", replyMsg, t_last.UnixMilli()-t_begin.UnixMilli())
	connWrite.Close()
}

func main() {
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
		test()
	}
}
