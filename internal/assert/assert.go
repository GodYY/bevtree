package assert

import "log"

func True(b bool, msg interface{}) {
	if !b {
		log.Panic(msg)
	}
}

func Truef(b bool, f string, args ...interface{}) {
	if !b {
		log.Panicf(f, args...)
	}
}

func False(b bool, msg interface{}) {
	True(!b, msg)
}

func Falsef(b bool, f string, args ...interface{}) {
	Truef(!b, f, args...)
}

func Equal(a, b interface{}, msg interface{}) {
	True(a == b, msg)
}

func Equalf(a, b interface{}, f string, args ...interface{}) {
	Truef(a == b, f, args...)
}

func NotEqual(a, b interface{}, msg interface{}) {
	True(a != b, msg)
}

func NotEqualf(a, b interface{}, f string, args ...interface{}) {
	Truef(a != b, f, args...)
}

func Nil(i interface{}, msg interface{}) {
	True(i == nil, msg)
}

func Nilf(i interface{}, f string, args ...interface{}) {
	Truef(i == nil, f, args...)
}

func NotNil(i interface{}, msg interface{}) {
	True(i != nil, msg)
}

func NotNilf(i interface{}, f string, args ...interface{}) {
	Truef(i != nil, f, args...)
}

func NotNilArg(arg interface{}, name string) {
	if arg == nil {
		log.Panicf("argument \"%s\" nil", name)
	}
}
