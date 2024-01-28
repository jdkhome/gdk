package common

var cleanupRegister = make([]func(), 0)

func RegisterCleanup(fun func()) {
	cleanupRegister = append(cleanupRegister, fun)
}

func Cleanup() {
	for _, fun := range cleanupRegister {
		fun()
	}
}
