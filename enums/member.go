package enums

type iMember[V any] interface {
	~struct {
		Code  string
		Value *V
	}
}
type Member[V any] struct {
	Code  string
	Value *V
}
