package enums

import "testing"

type testMember Member[Value]

func TestBuilder_Basic(t *testing.T) {
	b := NewBuilder[testMember]()
	red := b.Add("red", NewEnumValueWithDesc("红色", "红颜色"))
	blue := b.Add("blue", NewEnumValue("蓝色"))

	e := b.Build()

	if got, ok := e.GetByCode("red"); !ok || got != red {
		t.Errorf("GetByCode(red) = %v, %v", got, ok)
	}
	if got, ok := e.GetByCode("blue"); !ok || got != blue {
		t.Errorf("GetByCode(blue) = %v, %v", got, ok)
	}
	if !e.CheckMember(blue) {
		t.Error("blue 应为成员")
	}
	if e.CheckMember(testMember{Code: "green"}) {
		t.Error("green 不应为成员")
	}
	if _, ok := e.GetByCode("green"); ok {
		t.Error("GetByCode(green) 应返回 ok=false")
	}
	if len(e.Members()) != 2 {
		t.Errorf("Members() 长度 = %d, want 2", len(e.Members()))
	}
}

func TestBuilder_AddDuplicatePanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("重复添加相同 code 应 panic")
		}
	}()

	b := NewBuilder[testMember]()
	b.Add("red", Value{})
	b.Add("red", Value{})
}

func TestBuilder_AddAfterBuildPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Build 之后继续 Add 应 panic")
		}
	}()

	b := NewBuilder[testMember]()
	b.Add("red", Value{})
	b.Build()
	b.Add("blue", Value{})
}

func TestNewEnumValue(t *testing.T) {
	v := NewEnumValue("名称")
	if v.Name != "名称" {
		t.Errorf("Name = %q, want 名称", v.Name)
	}
	if v.Desc != "" {
		t.Errorf("Desc = %q, want empty", v.Desc)
	}
}

func TestNewEnumValueWithDesc(t *testing.T) {
	v := NewEnumValueWithDesc("名称", "描述")
	if v.Name != "名称" {
		t.Errorf("Name = %q, want 名称", v.Name)
	}
	if v.Desc != "描述" {
		t.Errorf("Desc = %q, want 描述", v.Desc)
	}
}
