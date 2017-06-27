package main

//socket代理
//只支持socket5
//只支持tcp
//不支持认证

import (
	"bytes"
	"encoding/binary"
	"net"
	"strconv"

	"github.com/hsyan2008/go-logger/logger"
)

func startSocket5(config Config) {
	if config.Addr == "" {
		logger.Warn("no addr")
		return
	}
	ln, err := net.Listen("tcp", config.Addr)
	if err != nil {
		logger.Warn("socket5 listen error:", err)
	}
	logger.Info("start socket5 listen ", config.Addr, "overssh", config.Overssh)

	for {
		con, err := ln.Accept()
		if err != nil {
			continue
		}
		logger.Debug("accept connect")
		go handSocket5(con, config.Overssh)
	}
}

func handSocket5(con net.Conn, overssh bool) {
	var buf []byte
	var err error

	//client发送请求来协商版本和认证方法
	buf = readLen(con, 1+1+255)

	if buf[0] != 0x05 {
		logger.Warn("只支持V5版本")
		_ = con.Close()
		return
	}

	//回应版本和认证方法
	_, _ = con.Write([]byte{0x05, 0x00})

	//请求目标地址
	buf = readLen(con, 4)
	cmd := buf[1]
	switch cmd {
	case 0x01: //tcp
	case 0x02: //bind
		logger.Warn("不支持BIND")
		_, _ = con.Write([]byte{0x05, 0x02, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
		_ = con.Close()
		return
	case 0x03: //udp
		logger.Warn("不支持UDP")
		_, _ = con.Write([]byte{0x05, 0x02, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
		_ = con.Close()
		return
	}
	atyp := buf[3]
	var host string
	var port uint16
	buf = readLen(con, 1024)
	switch atyp {
	case 0x01: //ipv4地址，php代码可以测试
		host = net.IP(buf[:4]).String()
	case 0x03: //域名，firefox浏览器下可以测试
		host = string(buf[1 : len(buf)-2])
	case 0x04: //ipv6地址
		logger.Warn("不支持ipv6")
		_, _ = con.Write([]byte{0x05, 0x02, 0x00, atyp, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
		_ = con.Close()
		return
	}
	_ = binary.Read(bytes.NewReader(buf[len(buf)-2:]), binary.BigEndian, &port)

	conn, err := dial(host+":"+strconv.Itoa(int(port)), overssh)
	if err != nil {
		logger.Warn(host, err)
		// _, _ = con.Write([]byte{0x05, 0x06, 0x00, atyp})
		_ = con.Close()
		return
	}
	logger.Info(host, port, "连接建立成功")

	_, _ = con.Write([]byte{0x05, 0x00, 0x00, atyp})
	//把地址写回去
	_, _ = con.Write(buf)

	go copyNet(con, conn)
	go copyNet(conn, con)
}

func readLen(con net.Conn, len int) (buf []byte) {
	buf = make([]byte, len)

	n, _ := con.Read(buf)

	return buf[:n]
}