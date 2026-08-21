//go:build diinject

package example

import "github.com/jdkhome/gdk/di"

func InitHandlers() []Handler {
	di.Build(HandlerSet)
	return nil
}

func InitUserRepo() Repository[User] {
	di.Build(RepoSet)
	return nil
}

func InitBox() *Box[User] {
	di.Build(BoxSet)
	return nil
}

func InitStructBox() *Box[User] {
	di.Build(BoxStructSet)
	return nil
}
