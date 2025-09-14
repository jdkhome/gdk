package enums

type Value struct {
	Name string // 名称
	Desc string // 描述
}

func NewEnumValue(name string) Value {
	return Value{
		Name: name,
	}
}

func NewEnumValueWithDesc(name, desc string) Value {
	return Value{
		Name: name,
		Desc: desc,
	}
}
