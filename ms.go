package main

import (
	"bytes"
	"database/sql"
	"encoding/binary"
	"flag"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"io/ioutil"
	"net"
	"strconv"
	"strings"
	"time"
)

var listen = flag.String("listen", "0.0.0.0", "ip address to listen")
var port = flag.Int("port", 27010, "port")
var useServerFile = flag.Bool("use-file", true, "use file containing ip address list")
var serverFile = flag.String("file", "servers.txt", "path to file")
var useDB = flag.Bool("use-db", false, "use database to fetch ip address list")
var DBType = flag.String("db-type", "mysql", "database type")
var DBURL = flag.String("db-url", "dbuser:dbpass@tcp(127.0.0.1:3306)/dbname", "database connection url")
var DBQuery = flag.String("db-query", "SELECT address FROM servers", "database query to fetch ip address list")
var useBanlist = flag.Bool("use-banlist", true, "use banlist to filter from ip address list")
var BanlistFile = flag.String("banlist-file", "banlist.txt", "banlist")

var (
	server_list []string
)

func WriteToServerList() {
	for {
		server_list = GetServerList()
		fmt.Println("checked")
		time.Sleep(60 * time.Second)
	}
}

func GetServerListDB() []string {
	var serverlist []string
	db, err := sql.Open(*DBType, *DBURL)
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
	rows, err := db.Query(*DBQuery)
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
	file, err := ioutil.ReadFile(*BanlistFile)
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
	if *useDB == true {
		serverlist = GetServerListDB()
	}
	if *useServerFile == true {
		file, err := ioutil.ReadFile(*serverFile)
		if err != nil {
			fmt.Println(err)
		}
		serverlist = append(serverlist, strings.Split(string(file), "\n")...)
	}
	if *useBanlist {
		serverlist = FilterBanlist(serverlist)
	}
	return serverlist
}

func main() {
	flag.Parse()
	ip := net.ParseIP(*listen)
	listener, err := net.ListenUDP("udp", &net.UDPAddr{IP: ip, Port: *port})
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
