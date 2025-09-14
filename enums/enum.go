package enums

import "sync"

type Enum[M iMember[V], V any] struct {
	keyOfMemberMap *sync.Map // code => *M
	members        []M
}

func (e *Enum[M, V]) Members() []M {
	return e.members
}

func (e *Enum[M, V]) CheckMember(m M) bool {
	code := Member[V](m).Code
	_, ok := e.keyOfMemberMap.Load(code)
	return ok
}

func (e *Enum[M, V]) GetByCode(code string) (M, bool) {
	if member, ok := e.keyOfMemberMap.Load(code); ok {
		return member.(M), true
	}
	return M{Code: code}, false
}
