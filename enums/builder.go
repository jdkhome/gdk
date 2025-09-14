package enums

import "sync"

type Builder[M iMember[V], V any] struct {
	finished bool      // 代表已经完成构建
	members  *sync.Map // code=>*M
}

func NewBuilder[M iMember[V], V any]() *Builder[M, V] {
	return &Builder[M, V]{
		finished: false,
		members:  &sync.Map{},
	}
}

func (b *Builder[M, V]) Add(code string, value V) M {
	if b.finished {
		panic("enum builder already finished!")
	}

	if _, ok := b.members.Load(code); ok {
		panic("enum code already exists!")
	}

	member := M{
		Code:  code,
		Value: &value,
	}
	b.members.Store(code, member)
	return member
}

func (b *Builder[M, V]) Build() Enum[M, V] {
	b.finished = true
	result := make([]M, 0)
	b.members.Range(func(k, v any) bool {
		result = append(result, v.(M))
		return true
	})
	return Enum[M, V]{
		keyOfMemberMap: b.members,
		members:        result,
	}
}
