// Program Detailed-Ping-Go is used for collectiog ping's results
package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/logrusorgru/aurora/v4"
	"github.com/mattn/go-colorable"
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

// PacketsLog used for printing final results
type PacketsLog struct {
	allPacketsSecond     int
	allPacketsMin        int
	allPacketsHour       int
	allPackets3Hour      int
	allPacketsAll        int
	droppedPacketsSecond int
	droppedPacketsMin    int
	droppedPacketsHour   int
	droppedPackets3Hour  int
	droppedPacketsAll    int
	avgLatencySecond     int
	avgLatencyMinute     int
	avgLatencyHour       int
	avgLatency3Hour      int
	avgLatencyAll        int
}

var p PacketsLog

/* For more readable code */
const second = 1
const minute = 60
const hour = 3600

/* Sets by arguments or have default values */
var ipAdr = ""
var timeout = 300

/* Logging options */
var logSecondEnabled = false
var logMinuteEnabled = false
var logHourEnabled = false
var log3HourEnabled = false
var logShowPacketsCount = false
var logInterval = second

/* Used for collect second's ping data */
var secondsPassed int
var queueMinute Queue
var queueHour Queue
var queue3Hour Queue

/* Used for processing ping data */
var dataSecond map[int]int
var statsSecond map[int]int
var statsMinute map[int]int
var statsHour map[int]int
var stats3Hour map[int]int
var statsAll map[int]int

/* Used for logging */
var mu sync.Mutex
var out io.Writer

var seq = 1

// colorizeLoss used for colorize loss value by some rules
func colorizeLoss(value float64) string {
	if value < 1 {
		return aurora.Sprintf(aurora.Bold("%.2f%%"), aurora.Green(value))
	} else if value < 5 {
		return aurora.Sprintf(aurora.Bold("%.2f%%"), aurora.Yellow(value))
	} else {
		return aurora.Sprintf(aurora.Bold("%.2f%%"), aurora.Red(value))
	}

}

// colorizeLatency used for colorize latency value by some rules
func colorizeLatency(value int, validPacketsCount int) string {
	if validPacketsCount == 0 {
		return aurora.Sprintf(aurora.Bold("%s ms"), aurora.Red("--"))
	}
	if value < 60 {
		return aurora.Sprintf(aurora.Bold("%d ms"), aurora.Green(value))
	} else if value < 120 {
		return aurora.Sprintf(aurora.Bold("%d ms"), aurora.Yellow(value))
	} else {
		return aurora.Sprintf(aurora.Bold("%d ms"), aurora.Red(value))
	}
}

// firstCommaPrint is used for detection of first item
func firstCommaPrint(firstOut *bool) string {
	if *firstOut {
		*firstOut = false
		return ""
	}
	return ", "
}

// printDropValue colorize and print drop at right order
func printDropValue(firstOut *bool, textInterval string, droppedPackets int, allPackets int) string {
	msg := ""
	msg += firstCommaPrint(firstOut)
	msg += (textInterval + ": ")
	stats := 100.
	if allPackets > 0 {
		stats = 100 * float64(droppedPackets) / float64(allPackets)
	}
	msg += colorizeLoss(stats)
	if logShowPacketsCount {
		msg += fmt.Sprintf(" (%d of %d)", droppedPackets, allPackets)
	}
	return msg
}

// printLatencyValue colorize and print latency at right order
func printLatencyValue(firstOut *bool, textInteval string, avgLatency int, validPackets int) string {
	msg := ""
	msg += firstCommaPrint(firstOut)
	msg += (textInteval + ": ")
	msg += colorizeLatency(avgLatency, validPackets)
	return msg
}

// printMsg prints one line of data including packet drops and latency
func printMsg(strTime string) {
	var firstOut bool
	firstOut = true
	var msg = ""
	msg += "[" + strTime + "]\t"
	msg += "Loss: "
	if logSecondEnabled {
		msg += printDropValue(&firstOut, "sec", p.droppedPacketsSecond, p.allPacketsSecond)
	}
	if logMinuteEnabled {
		msg += printDropValue(&firstOut, "min", p.droppedPacketsMin, p.allPacketsMin)
	}
	if logHourEnabled {
		msg += printDropValue(&firstOut, "hour", p.droppedPacketsHour, p.allPacketsHour)
	}
	if log3HourEnabled {
		msg += printDropValue(&firstOut, "3 hours", p.droppedPackets3Hour, p.allPackets3Hour)
	}
	msg += printDropValue(&firstOut, "all", p.droppedPacketsAll, p.allPacketsAll)

	firstOut = true
	msg += "\tLatency: "
	if logSecondEnabled {
		msg += printLatencyValue(&firstOut, "sec", p.avgLatencySecond, p.allPacketsSecond-p.droppedPacketsSecond)
	}
	if logMinuteEnabled {
		msg += printLatencyValue(&firstOut, "min", p.avgLatencyMinute, p.allPacketsMin-p.droppedPacketsMin)
	}
	if logHourEnabled {
		msg += printLatencyValue(&firstOut, "hour", p.avgLatencyHour, p.allPacketsHour-p.droppedPacketsHour)
	}
	if log3HourEnabled {
		msg += printLatencyValue(&firstOut, "3 hours", p.avgLatency3Hour, p.allPackets3Hour-p.droppedPackets3Hour)
	}
	msg += printLatencyValue(&firstOut, "all", p.avgLatencyAll, p.allPacketsAll-p.droppedPacketsAll)

	fmt.Fprintln(out, msg)
}

// remOldStats removes no longer actual second from stats range
func remOldStats(stats map[int]int, q *Queue) {
	remOldSec := q.Pop()
	for k, v := range remOldSec {
		if stats[k] > 0 {
			stats[k] -= v
		}
	}
}

// calcStats calculate packet drop and avg latency
func calcStats(stats map[int]int, droppedPackets *int, allPackets *int, avgLatency *int) {
	for k, v := range stats {
		if k >= timeout {
			*droppedPackets += v
		} else {
			*avgLatency += (k * v)
		}
		*allPackets += v
	}
	if (*allPackets - *droppedPackets) != 0 {
		*avgLatency /= ((*allPackets) - (*droppedPackets))
	}
}

// clearPacketLogs perform clean of all printable data
func clearPacketLogs() {
	p.allPacketsSecond = 0
	p.allPacketsMin = 0
	p.allPacketsHour = 0
	p.allPackets3Hour = 0
	p.allPacketsAll = 0
	p.droppedPacketsSecond = 0
	p.droppedPacketsMin = 0
	p.droppedPacketsHour = 0
	p.droppedPackets3Hour = 0
	p.droppedPacketsAll = 0
	p.avgLatencySecond = 0
	p.avgLatencyMinute = 0
	p.avgLatencyHour = 0
	p.avgLatency3Hour = 0
	p.avgLatencyAll = 0
}

// log performs stats calculation per second and print results per interval
func log() {
	tIntervalCheck := time.Now()
	tLast := time.Now()
	for {
		tNow := time.Now()
		if tNow.Unix()-tLast.Unix() >= int64(second) {
			mu.Lock()
			secondsPassed++
			statsSecond = dataSecond
			queueMinute.Push(dataSecond)
			queueHour.Push(dataSecond)
			queue3Hour.Push(dataSecond)
			if secondsPassed >= minute {
				remOldStats(statsMinute, &queueMinute)
			}
			if secondsPassed >= hour {
				remOldStats(statsHour, &queueHour)
			}
			if secondsPassed >= 3*hour {
				remOldStats(stats3Hour, &queue3Hour)
			}
			dataSecond = nil
			mu.Unlock()
			tLast = tNow
		}
		if tNow.Unix()-tIntervalCheck.Unix() >= int64(logInterval) {
			mu.Lock()
			strTime := tNow.Format(time.Stamp)
			clearPacketLogs()
			calcStats(statsSecond, &p.droppedPacketsSecond, &p.allPacketsSecond, &p.avgLatencySecond)
			calcStats(statsMinute, &p.droppedPacketsMin, &p.allPacketsMin, &p.avgLatencyMinute)
			calcStats(statsHour, &p.droppedPacketsHour, &p.allPacketsHour, &p.avgLatencyHour)
			calcStats(stats3Hour, &p.droppedPackets3Hour, &p.allPackets3Hour, &p.avgLatency3Hour)
			calcStats(statsAll, &p.droppedPacketsAll, &p.allPacketsAll, &p.avgLatencyAll)
			printMsg(strTime)
			tIntervalCheck = tNow
			mu.Unlock()
		}
	}
}

// test sends ICMP packet and retrieve answer, collecting latency and drop results
func test() error {
	connWrite, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		return err
	}
	// default timeout = 300 ms
	connWrite.SetDeadline(time.Now().Add(time.Millisecond * time.Duration(timeout)))
	ipAddr, err := net.ResolveIPAddr("ip4", ipAdr)
	if err != nil {
		return err
	}
	msg := icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.Echo{
			ID:   os.Getpid() & 0xffff,
			Seq:  seq,
			Data: []byte("0 1 2 3 4 5 6 7 8 9 10"),
		},
	}
	// required, otherwise strange behavior occurs
	seq++
	msgBytes, err := msg.Marshal(nil)
	if err != nil {
		connWrite.Close()
		return err
	}
	tBegin := time.Now()
	if _, err := connWrite.WriteTo(msgBytes, ipAddr); err != nil {
		connWrite.Close()
		return err
	}
	buf := make([]byte, 50)
	n, _, errRead := connWrite.ReadFrom(buf)
	tLast := time.Now()
	latency := tLast.UnixMilli() - tBegin.UnixMilli()
	if n == 0 {
		connWrite.Close()
		return nil
	}
	if errRead != nil {
		// i/o timeout, but may be other errors
		connWrite.Close()
		return errRead
	}
	mu.Lock()
	if dataSecond == nil {
		dataSecond = make(map[int]int)
	}
	// if no timeout, write ping data
	dataSecond[int(latency)]++
	statsHour[int(latency)]++
	stats3Hour[int(latency)]++
	statsMinute[int(latency)]++
	statsAll[int(latency)]++
	mu.Unlock()
	connWrite.Close()
	return nil
}

// printHelp used for incorrect call of dping.go
func printHelp() {
	fmt.Println("USAGE: dping IPv4 [arguments]")
	fmt.Println("Available arguments: ")
	fmt.Println("-t [msec] or --timeout [msec]\tSet timeout for packets.\t\t(default msec = 300)")
	fmt.Println("-i s/m/h or --interval s/m/h\tSet logging interval to sec/min/hour.\t(default i = s)")
	fmt.Println("-s or --second\t\t\tEnable logging second drop stats. \t(default enabled, if i = s)")
	fmt.Println("-m or --min\t\t\tEnable logging minute drop stats. \t(default enabled, if i = m)")
	fmt.Println("-h or --hour\t\t\tEnable logging hour drop stats. \t(default enabled, if i = h)")
	fmt.Println("-3h or --3hour\t\t\tEnable logging 3 hour drop stats \t(default disabled)")
	fmt.Println("-p or --packets\t\t\tEnable logging packets count stats. \t(default disabled)")
}

func main() {
	if runtime.GOOS == "windows" {
		out = colorable.NewColorableStdout()
	} else {
		out = os.Stdout
	}
	argsGiven := os.Args[1:]
	if len(argsGiven) < 1 {
		printHelp()
		return
	}
	if strings.Count(argsGiven[0], ".") != 3 {
		printHelp()
		return
	}
	ipAdr = argsGiven[0]
	for i := 1; i < len(argsGiven); i++ {
		switch argsGiven[i] {
		case "-t":
			fallthrough
		case "--timeout":
			if i+1 < len(argsGiven) {
				var err error
				timeout, err = strconv.Atoi(argsGiven[i+1])
				if err != nil {
					fmt.Println(err.Error())
					return
				}
				i++
			} else {
				fmt.Println(argsGiven[i], "requeires an argument")
				printHelp()
				return
			}
		case "-i":
			fallthrough
		case "--interval":
			if i+1 < len(argsGiven) {
				if argsGiven[i+1] == "s" {
					logInterval = second
				} else if argsGiven[i+1] == "m" {
					logInterval = minute
				} else if argsGiven[i+1] == "h" {
					logInterval = hour
				} else {
					fmt.Println(argsGiven[i], "accepts only 's', 'm' or 'h' argument (second, minute or hour)")
					return
				}
				i++
			} else {
				fmt.Println(argsGiven[i], "requeires an argument")
				printHelp()
				return
			}
		case "-s":
			fallthrough
		case "--second":
			logSecondEnabled = true
		case "-m":
			fallthrough
		case "--min":
			logMinuteEnabled = true
		case "-h":
			fallthrough
		case "--hour":
			logHourEnabled = true
		case "-3h":
			fallthrough
		case "--3hour":
			log3HourEnabled = true
		case "-p":
			fallthrough
		case "--packets":
			logShowPacketsCount = true
		default:
			fmt.Println("Unrecongnized argument:", argsGiven[i])
			printHelp()
			return
		}
	}
	if logInterval == second {
		logSecondEnabled = true
	} else if logInterval == minute {
		logMinuteEnabled = true
	} else if logInterval == hour {
		logHourEnabled = true
	}

	queueMinute = Queue{}
	queueHour = Queue{}
	queue3Hour = Queue{}

	secondsPassed = 0
	dataSecond = make(map[int]int)
	statsHour = make(map[int]int)
	stats3Hour = make(map[int]int)
	statsMinute = make(map[int]int)
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
