package enums

import "testing"

type color Member[colorValue]
type colorValue struct {
	Value
	Hello func()
}

var (
	colorEnumBuilder = NewBuilder[color]()
	color_Red        = colorEnumBuilder.Add("red", colorValue{Value: NewEnumValue("红色"), Hello: func() {
		println("你好红色")
	}})
	color_Yello = colorEnumBuilder.Add("yello", colorValue{Value: NewEnumValue("黄色"), Hello: func() {
		println("你好黄啊")
	}})
	colors = colorEnumBuilder.Build()
)

func TestEnum(t *testing.T) {
	for _, color := range colors.members {
		color.Value.Hello()
	}

	redEnum, ok := colors.GetByCode("red")
	if !ok {
		t.Errorf("red enum miss!")
	}
	if redEnum != color_Red {
		t.Errorf("red enum not equal!")
	}
	if ok := colors.CheckMember(redEnum); !ok {
		t.Errorf("red is not member!")
	}

}
