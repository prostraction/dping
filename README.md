# Detailed Ping Go (Go 1.17)
Simple ICMP ping tester with packet loss shown while running

To do:
- Set timeout from argv
- Set log timing from argv
- Set ip from argv
- More nice view of log
- Logging latency (ms)
- Logging "---" instead hour and 3-hour if program running less then hour


Configure:
```
git clone https://github.com/prostraction/Detailed-Ping-Go
cd Detailed-Ping-Go
go mod init main.go
go mod tidy
```

Run:
```
go run main.go
```

ICMP connection may require superuser privileges. If you encountered an error like `listen ip4:icmp 0.0.0.0: socket: operation not permitted` try:

```
sudo go run main.go
```
