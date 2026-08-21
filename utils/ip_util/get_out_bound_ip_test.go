package ip_util

import (
	"net"
	"testing"
)

func TestGetOutBoundIP(t *testing.T) {
	ip, err := GetOutBoundIP()
	if err != nil {
		t.Skipf("网络不可用，跳过测试: %v", err)
	}
	if net.ParseIP(ip) == nil {
		t.Errorf("GetOutBoundIP 返回非法 IP: %q", ip)
	}
}

func TestGetIpHash(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Skipf("网络不可用，跳过测试: %v", r)
		}
	}()

	hash := GetIpHash()
	if len(hash) != 6 {
		t.Errorf("GetIpHash 长度 = %d, want 6", len(hash))
	}
}
