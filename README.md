# Detailed Ping Go (Go 1.17+)

[![Codacy Badge](https://app.codacy.com/project/badge/Grade/0b621290c3ad494bae3b8524fe8e55ee)](https://app.codacy.com/gh/prostraction/Detailed-Ping-Go/dashboard?utm_source=gh&utm_medium=referral&utm_content=&utm_campaign=Badge_grade)

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

Usage: 
```
dping IPv4 [arguments]
```

Available arguments:
-   `-t [msec]` or `--timeout [msec]`   Set timeout for packets.              (default msec = `300`)
-   `-i s/m/h` or `--interval s/m/h`    Set logging interval to sec/min/hour. (default `i = m`)
-   `-s` or `--second`                  Enable logging second drop stats.     (default enabled, if `i = s`)
-   `-m` or `--min`                     Enable logging minute drop stats.     (default enabled, if `i = m`)
-   `-h` or `--hour`                    Enable logging hour drop stats.       (default enabled, if `i = h`)
-   `-3h` or `--3hour`                  Enable logging 3 hour drop stats      (default disabled)
-   `-p` or `--packets`                 Enable logging packets count stats.   (default disabled)



![dping1](https://user-images.githubusercontent.com/47314760/228662319-5ebdf4c5-61ef-49d2-a778-c048cc980aad.PNG)
