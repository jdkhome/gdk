package ip_util

import (
	"github.com/duke-git/lancet/v2/cryptor"
	"net"
	"strings"
)

func GetOutBoundIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:53")
	if err != nil {
		return "", err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	ip := strings.Split(localAddr.String(), ":")[0]
	return ip, nil
}

func GetIpHash() string {
	ip, err := GetOutBoundIP()
	if err != nil {
		panic(err)
	}
	return cryptor.Md5String(ip)[:6]
}
