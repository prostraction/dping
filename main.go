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

var ip_adr = "127.0.0.1"

var seconds_passed int
var queue_min Queue
var queue_hour Queue
var queue_3_hour Queue
var stats_second map[int]int
var stats_min map[int]int
var stats_hour map[int]int
var stats_3_hour map[int]int
var stats_all map[int]int

// var seq int
var mu sync.Mutex

func log() {
	t_min_check := time.Now()
	t_last := time.Now()
	for {
		t_now := time.Now()
		if t_now.Second()-t_last.Second() != 0 {
			mu.Lock()
			seconds_passed++

			queue_min.Push(stats_second)
			queue_hour.Push(stats_second)
			queue_3_hour.Push(stats_second)

			if seconds_passed >= 60 {
				rem_old_sec := queue_min.Pop()
				for k, v := range rem_old_sec {
					if stats_min[k] > 0 {
						stats_min[k] -= v
					}
				}
			}
			if seconds_passed >= 3600 {
				rem_old_sec := queue_hour.Pop()
				for k, v := range rem_old_sec {
					if stats_hour[k] > 0 {
						stats_hour[k] -= v
					}
				}
			}
			if seconds_passed >= 3*3600 {
				rem_old_sec := queue_3_hour.Pop()
				for k, v := range rem_old_sec {
					if stats_3_hour[k] > 0 {
						stats_3_hour[k] -= v
					}
				}
			}

			stats_second = nil
			mu.Unlock()
			t_last = t_now
		}
		if t_now.Minute()-t_min_check.Minute() != 0 {
			mu.Lock()
			strTime := t_now.Format(time.Stamp)
			all_packets_min := 0
			all_packets_hour := 0
			all_packets_3_hour := 0
			all_packets_all := 0

			dropped_packets_min := 0
			dropped_packets_hour := 0
			dropped_packets_3_hour := 0
			dropped_packets_all := 0

			for k, v := range stats_min {
				if k >= 150 {
					dropped_packets_min += v
				}
				all_packets_min += v
			}
			for k, v := range stats_hour {
				if k >= 150 {
					dropped_packets_hour += v
				}
				all_packets_hour += v
			}
			for k, v := range stats_3_hour {
				if k >= 150 {
					dropped_packets_3_hour += v
				}
				all_packets_3_hour += v
			}
			for k, v := range stats_all {
				if k >= 150 {
					dropped_packets_all += v
				}
				all_packets_all += v
			}
			if all_packets_min > 0 {
				fmt.Printf("[%s]    loss M: [%.2f%% (%d of %d)], H: [%.2f%% (%d of %d)], 3H: [%.2f%% (%d of %d)], ALL: [%.2f%% (%d of %d)]\n", strTime,
					100*float64(dropped_packets_min)/float64(all_packets_min),
					dropped_packets_min, all_packets_min,

					100*float64(dropped_packets_hour)/float64(all_packets_hour),
					dropped_packets_hour, all_packets_hour,

					100*float64(dropped_packets_3_hour)/float64(all_packets_3_hour),
					dropped_packets_3_hour, all_packets_3_hour,

					100*float64(dropped_packets_all)/float64(all_packets_all),
					dropped_packets_all, all_packets_all)
			}
			mu.Unlock()
			t_min_check = t_now
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
	ipAddr, err := net.ResolveIPAddr("ip4", ip_adr)
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

	t_begin := time.Now()

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
	t_last := time.Now()
	mu.Lock()
	if stats_second == nil {
		stats_second = make(map[int]int)
	}
	latency := t_last.UnixMilli() - t_begin.UnixMilli()
	stats_second[int(latency)]++
	stats_hour[int(latency)]++
	stats_3_hour[int(latency)]++
	stats_min[int(latency)]++
	stats_all[int(latency)]++
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
	queue_min = Queue{}
	queue_hour = Queue{}
	queue_3_hour = Queue{}

	seconds_passed = 0
	stats_second = make(map[int]int)
	stats_hour = make(map[int]int)
	stats_3_hour = make(map[int]int)
	stats_min = make(map[int]int)
	stats_all = make(map[int]int)
	go log()
	for {
		test()
	}
}
