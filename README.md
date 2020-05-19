# ICMP Echo CLI written in Go


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

## Examples

```
sudo ./goping google.com 35 ip6
sudo ./goping reddit.com 20 ip6
```
