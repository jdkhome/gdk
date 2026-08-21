package model

import "testing"

func TestIPVersions(t *testing.T) {
	if len(IPVersions.Members()) != 2 {
		t.Errorf("Members() 长度 = %d, want 2", len(IPVersions.Members()))
	}

	if !IPVersions.CheckMember(IPVersion_V4) {
		t.Error("IPVersion_V4 应为成员")
	}

	if got, ok := IPVersions.GetByCode("ipv6"); !ok || got != IPVersion_V6 {
		t.Errorf("GetByCode(ipv6) = %v, %v", got, ok)
	}

	if _, ok := IPVersions.GetByCode("ipv8"); ok {
		t.Error("GetByCode(ipv8) 应返回 ok=false")
	}
}
