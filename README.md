# ICMP Echo CLI written in Go

See ss_1.png and ss_2.png for examples

## Installation


```
go install
go build goping.go
```

## Usage
UNIX: The ICMP protocol requires sudo privileges 
```
sudo ./goping [address] [Preferred TTL] [protocol (ip4 or ip6)]
```
Windows: Not tested
```
goping [address] [Preferred TTL] [protocol (ip4 or ip6)]