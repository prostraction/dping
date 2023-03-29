# Detailed Ping Go (Go 1.17+)
Simple ICMP ping tester with packet loss shown while running.

Get project:
```
git clone https://github.com/prostraction/Detailed-Ping-Go
cd Detailed-Ping-Go
```

Run:
```
go run dping.go
```

Build:
```
go build -o bin/dping.exe
```

ICMP connection may require superuser privileges. If you encountered an error like `listen ip4:icmp 0.0.0.0: socket: operation not permitted` try:
```
sudo go run dping.go
```

USAGE: dping IPv4 [arguments]
Available arguments:
-   `-t [msec]` or `--timeout [msec]`   Set timeout for packets.              (default msec = `300`)
-   `-i s/m/h` or `--interval s/m/h`    Set logging interval to sec/min/hour. (default `i = m`)
-   `-s` or `--second`                  Enable logging second drop stats.     (default enabled, if `i = s`)
-   `-m` or `--min`                     Enable logging minute drop stats.     (default enabled, if `i = m`)
-   `-h` or `--hour`                    Enable logging hour drop stats.       (default enabled, if `i = h`)
-   `-3h` or `--3hour`                  Enable logging 3 hour drop stats      (default disabled)
-   `-p` or `--packets`                 Enable logging packets count stats.   (default disabled)
