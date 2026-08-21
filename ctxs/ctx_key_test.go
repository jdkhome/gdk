package ctxs

import "testing"

func TestKeys(t *testing.T) {
	if !Keys.CheckMember(Trace) {
		t.Error("Trace 应为 Keys 的成员")
	}

	got, ok := Keys.GetByCode("CTX#TRACE")
	if !ok {
		t.Fatal("GetByCode(CTX#TRACE) 未找到")
	}
	if got != Trace {
		t.Errorf("GetByCode(CTX#TRACE) = %v, want %v", got, Trace)
	}

	if len(Keys.Members()) == 0 {
		t.Error("Keys.Members() 不应为空")
	}

	if _, ok := Keys.GetByCode("NOT_EXIST"); ok {
		t.Error("GetByCode(NOT_EXIST) 应返回 ok=false")
	}
	if Keys.CheckMember(Key{Code: "NOT_EXIST"}) {
		t.Error("CheckMember(NOT_EXIST) 应为 false")
	}
}
