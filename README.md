# Detailed Ping Go (Go 1.17+)
Simple ICMP ping tester with packet loss shown while running.

Configure:
```
git clone https://github.com/prostraction/Detailed-Ping-Go
cd Detailed-Ping-Go
```

Run:
```
go run main.go
```

ICMP connection may require superuser privileges. If you encountered an error like `listen ip4:icmp 0.0.0.0: socket: operation not permitted` try:
```
sudo go run main.go
```
