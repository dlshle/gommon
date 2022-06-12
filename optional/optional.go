package optional

import "github.com/dlshle/gommon/errors"

var Empty = Of(nil)

type Optional interface {
	GetOrError() (interface{}, error)
	GetOrPanic() interface{}
	isPresent() bool
	ifPresent(func(interface{}))
	Filter(func(interface{}) bool) Optional
	Map(func(val interface{}) interface{}) Optional
	OrElse(interface{}) interface{}
	OrElseGet(func() interface{}) interface{}
	OrElsePanic(interface{}) interface{}
}

type optional struct {
	val interface{}
}

func (o *optional) GetOrError() (interface{}, error) {
	if o.val == nil {
		return nil, errors.Error("empty optional value")
	}
	return o.val, nil
}

func (o *optional) GetOrPanic() interface{} {
	val, err := o.GetOrError()
	if err != nil {
		panic(err)
	}
	return val
}

func (o *optional) isPresent() bool {
	return o.val != nil
}

func (o *optional) ifPresent(thenFunc func(interface{})) {
	if o.val == nil {
		return
	}
	thenFunc(o.val)
}

func (o *optional) Filter(filterFunc func(interface{}) bool) Optional {
	if o.val == nil {
		return Empty
	}
	if filterFunc(o.val) {
		return o
	}
	return Empty
}

func (o *optional) Map(mappingFunc func(val interface{}) interface{}) Optional {
	if o.val == nil {
		return Empty
	}
	return Of(mappingFunc(o.val))
}

func (o *optional) OrElse(val interface{}) interface{} {
	if o.val == nil {
		return val
	}
	return o.val
}

func (o *optional) OrElseGet(getFunc func() interface{}) interface{} {
	if o.val == nil {
		return getFunc()
	}
	return o.val
}

func (o *optional) OrElsePanic(panicVal interface{}) interface{} {
	if o.val == nil {
		panic("implement me")
	}
	return o.val
}

func Of(val interface{}) Optional {
	return &optional{
		val: val,
	}
}
