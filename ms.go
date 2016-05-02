package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"github.com/BurntSushi/toml"
	"io/ioutil"
	"net"
	"strconv"
	"strings"
)

type Config struct {
	Host string
	Port int
	File string
}

var (
	configFile = flag.String("config", "ms.cfg", "config file")
)

func main() {
	flag.Parse()
	var config Config

	if _, err := toml.DecodeFile(*configFile, &config); err != nil {
		fmt.Println(err)
		return
	}

	file, err := ioutil.ReadFile(config.File)
	if err != nil {
		fmt.Println(err)
		return
	}
	serverlist := strings.Split(string(file), "\n")

	ip := net.ParseIP(config.Host)

	listener, err := net.ListenUDP("udp", &net.UDPAddr{IP: ip, Port: config.Port})
	if err != nil {
		fmt.Println(err)
		return
	}

	data := make([]byte, 1024)

	for {
		n, remoteAddr, err := listener.ReadFromUDP(data)
		if err != nil {
			fmt.Printf("error during read: %s", err)
			return
		}

		buf := new(bytes.Buffer)

		binary.Write(buf, binary.LittleEndian, []byte{0xFF, 0xFF, 0xFF, 0xFF, 0x66, 0x0A})

		for _, server := range serverlist {
			host, port, err := net.SplitHostPort(server)
			if err != nil {
				continue
			}

			ip = net.ParseIP(host).To4()
			if ip == nil {
				continue
			}

			port_i, _ := strconv.Atoi(port)
			port_i16 := int16(port_i)
			port_o := port_i16<<8 | port_i16>>8

			binary.Write(buf, binary.LittleEndian, ip)
			binary.Write(buf, binary.LittleEndian, port_o)
		}

		binary.Write(buf, binary.LittleEndian, []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00})

		_, err = listener.WriteToUDP(buf.Bytes(), remoteAddr)

		if err != nil {
			fmt.Println(err)
		}

		fmt.Printf("<%s> %s\n", remoteAddr, data[:n])
	}
}
