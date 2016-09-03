package main

import (
	"bytes"
	"database/sql"
	"encoding/binary"
	"flag"
	"fmt"
	"github.com/BurntSushi/toml"
	_ "github.com/go-sql-driver/mysql"
	"io/ioutil"
	"net"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Host         string
	Port         int
	Use_file     bool
	Server_file  string
	Use_db       bool
	Db_type      string
	Db_url       string
	Db_query     string
	Use_banlist  bool
	Banlist_file string
}

var (
	configFile  = flag.String("config", "ms.cfg", "config file")
	config      Config
	server_list []string
)

func WriteToServerList() {
	for {
		server_list = GetServerList()
		fmt.Println("Checked")
		time.Sleep(60 * time.Second)
	}
}

func GetServerListDB() []string {
	var serverlist []string
	db, err := sql.Open(config.Db_type, config.Db_url)
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}
	defer db.Close()
	err = db.Ping()
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}
	rows, err := db.Query(config.Db_query)
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}
	defer rows.Close()
	var dbAddress string
	for rows.Next() {
		err := rows.Scan(&dbAddress)
		if err != nil {
			fmt.Println(err)
		}
		serverlist = append(serverlist, dbAddress)
	}
	err = rows.Err()
	if err != nil {
		fmt.Println(err)
		return nil
	}
	return serverlist
}

func FilterBanlist(serverlist []string) []string {
	file, err := ioutil.ReadFile(config.Banlist_file)
	if err != nil {
		fmt.Println(err)
		return serverlist
	}
	var new_serverlist []string
	banlist := strings.Split(string(file), "\n")
	for _, server := range serverlist {
		banned := false
		for _, bserver := range banlist {
			if bserver == server {
				banned = true
			}
		}
		if banned == false {
			new_serverlist = append(new_serverlist, server)
		}
	}
	return new_serverlist
}

func GetServerList() []string {
	var serverlist []string
	if config.Use_db == true {
		serverlist = GetServerListDB()
	}
	if config.Use_file == true {
		file, err := ioutil.ReadFile(config.Server_file)
		if err != nil {
			fmt.Println(err)
		}
		serverlist = append(serverlist, strings.Split(string(file), "\n")...)
	}
	if config.Use_banlist {
		serverlist = FilterBanlist(serverlist)
	}
	return serverlist
}

func main() {
	flag.Parse()
	if _, err := toml.DecodeFile(*configFile, &config); err != nil {
		fmt.Println(err)
		return
	}
	ip := net.ParseIP(config.Host)
	listener, err := net.ListenUDP("udp", &net.UDPAddr{IP: ip, Port: config.Port})
	if err != nil {
		fmt.Println(err)
		return
	}
	go WriteToServerList()
	data := make([]byte, 1024)
	for {
		n, remoteAddr, err := listener.ReadFromUDP(data)
		if err != nil {
			fmt.Printf("error during read: %s", err)
			return
		}
		buf := new(bytes.Buffer)
		binary.Write(buf, binary.LittleEndian, []byte{0xFF, 0xFF, 0xFF, 0xFF, 0x66, 0x0A})
		if strings.Contains(string(data[:n]), "0.0.0.0") {
			for _, server := range server_list {
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
		}
		_, err = listener.WriteToUDP(buf.Bytes(), remoteAddr)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Printf("<%s> %s\n", remoteAddr, data[:n])
	}
}
