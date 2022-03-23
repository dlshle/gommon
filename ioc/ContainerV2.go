package ioc

type SimpleFactory func() interface{}
type Factory func() (interface{}, error)
type SimpleDepFactory func(dependencies ...interface{}) interface{}
type DepFactory func(dependencies ...interface{}) (interface{}, error)

type DependencyManager interface {
	Singleton(factory SimpleFactory) error
	MaybeSingleton(factory Factory) error
	Prototype(factory SimpleFactory) error
	MaybePrototype(factory Factory) error
	Call(factory SimpleDepFactory) error
	MaybeCall(factory DepFactory) error
}

type dependencyManager struct {
}
