package ioc

import (
	"fmt"
	"reflect"
	"unsafe"

	"github.com/dlshle/gommon/utils"
)

const (
	TagInject          = "$inject"
	setterMethodPrefix = "Set"
)

// binding keeps a binding resolver and an instance (for singleton bindings).
type binding struct {
	resolver interface{} // resolver function that creates the appropriate implementation of the related abstraction
	instance interface{} // instance stored for reusing in singleton bindings
}

// resolve will create the appropriate implementation of the related abstraction
func (b *binding) resolve(c *Container) (interface{}, error) {
	if b.instance != nil {
		return b.instance, nil
	}
	var err error
	b.instance, err = c.invoke(b.resolver)
	return b.instance, err
}

// TypeContainer is the repository of bindings
type TypeContainer map[reflect.Type]*binding
type IdContainer map[string]*binding
type Container struct {
	typeContainer TypeContainer
	idContainer   IdContainer
}

// New creates a new instance of TypeContainer
func New() *Container {
	return &Container{
		typeContainer: make(TypeContainer),
		idContainer:   make(IdContainer),
	}
}

func (c *Container) checkAndGetResolverReflect(resolver interface{}) (reflect.Type, error) {
	reflectedResolver := reflect.TypeOf(resolver)
	if reflectedResolver.Kind() != reflect.Func {
		return nil, fmt.Errorf("container: the resolver must be a function")
	}
	return reflectedResolver, nil
}

// bindById will map an id to a concrete
func (c *Container) bindById(id string, resolver interface{}) error {
	reflectedResolver, err := c.checkAndGetResolverReflect(resolver)
	if err != nil {
		return err
	}
	if reflectedResolver.NumOut() != 1 {
		return fmt.Errorf("container: the resolver for id binding must has only 1 output value")
	}
	c.idContainer[id] = &binding{resolver: resolver}

	return nil
}

// bindByType will map an abstraction to a concrete
func (c *Container) bindByType(resolver interface{}) error {
	reflectedResolver, err := c.checkAndGetResolverReflect(resolver)
	if err != nil {
		return err
	}
	for i := 0; i < reflectedResolver.NumOut(); i++ {
		c.typeContainer[reflectedResolver.Out(i)] = &binding{resolver: resolver}
	}

	return nil
}

// invoke will call the given function and return its returned value.
// It only works for functions that return a single value.
func (c *Container) invoke(function interface{}) (interface{}, error) {
	args, err := c.arguments(function)
	if err != nil {
		return nil, err
	}

	return reflect.ValueOf(function).Call(args)[0].Interface(), nil
}

// arguments will return resolved arguments of the given function.
func (c *Container) arguments(function interface{}) ([]reflect.Value, error) {
	reflectedFunction := reflect.TypeOf(function)
	argumentsCount := reflectedFunction.NumIn()
	arguments := make([]reflect.Value, argumentsCount)

	for i := 0; i < argumentsCount; i++ {
		abstraction := reflectedFunction.In(i)

		if concrete, ok := c.typeContainer[abstraction]; ok {
			instance, err := concrete.resolve(c)
			if err != nil {
				return nil, err
			}

			arguments[i] = reflect.ValueOf(instance)
		} else {
			return nil, fmt.Errorf("container: no concrete found for: " + abstraction.String())
		}
	}

	return arguments, nil
}

// Singleton will bindByType an abstraction to a concrete for further singleton resolves.
// It takes a resolver function which returns the concrete and its return type matches the abstraction (interface).
// The resolver function can have arguments of abstraction that have bound already in TypeContainer.
func (c *Container) Singleton(resolver interface{}) error {
	return c.bindByType(resolver)
}

func (c *Container) RemoveByType(holder interface{}) error {
	reflectType := reflect.TypeOf(holder)
	if reflectType.Kind() == reflect.Func {
		return fmt.Errorf("type can not be of Func")
	}
	delete(c.typeContainer, reflectType)
	return nil
}

func (c *Container) RegisterSingleton(id string, resolver interface{}) error {
	return c.bindById(id, resolver)
}

// Reset will reset the container and remove all the existing bindings.
func (c Container) Reset() {
	for k := range c.typeContainer {
		delete(c.typeContainer, k)
	}
	for k := range c.idContainer {
		delete(c.idContainer, k)
	}
}

func (c *Container) GetById(id string) (interface{}, error) {
	concrete, ok := c.idContainer[id]
	if !ok {
		return nil, fmt.Errorf("can not find concrete by id " + id)
	}
	return concrete.resolve(c)
}

// Make will resolve the dependency and return a appropriate concrete of the given abstraction.
// It can take an abstraction (interface reference) and fill it with the related implementation.
// It also can takes a function (receiver) with one or more arguments of the abstractions (interfaces) that need to be
// resolved, TypeContainer will invoke the receiver function and pass the related implementations.
// Deprecated: Make is deprecated.
func (c *Container) Make(receiver interface{}) error {
	receiverType := reflect.TypeOf(receiver)
	if receiverType == nil {
		return fmt.Errorf("container: cannot detect type of the receiver")
	}

	if receiverType.Kind() == reflect.Ptr {
		return c.Bind(receiver)
	} else if receiverType.Kind() == reflect.Func {
		return c.Call(receiver)
	}

	return fmt.Errorf("container: the receiver must be either a reference or a callback")
}

// Call takes a function with one or more arguments of the abstractions (interfaces) that need to be
// resolved, TypeContainer will invoke the receiver function and pass the related implementations.
func (c *Container) Call(function interface{}) error {
	receiverType := reflect.TypeOf(function)
	if receiverType == nil {
		return fmt.Errorf("container: invalid function")
	}

	if receiverType.Kind() == reflect.Func {
		arguments, err := c.arguments(function)
		if err != nil {
			return err
		}

		reflect.ValueOf(function).Call(arguments)

		return nil
	}

	return fmt.Errorf("container: invalid function")
}

// Bind takes an abstraction (interface reference) and fill it with the related implementation.
func (c *Container) Bind(abstraction interface{}) error {
	receiverType := reflect.TypeOf(abstraction)
	if receiverType == nil {
		return fmt.Errorf("container: invalid abstraction")
	}

	if receiverType.Kind() == reflect.Ptr {
		elem := receiverType.Elem()

		if concrete, ok := c.typeContainer[elem]; ok {
			instance, err := concrete.resolve(c)
			if err != nil {
				return err
			}

			reflect.ValueOf(abstraction).Elem().Set(reflect.ValueOf(instance))

			return nil
		}

		return fmt.Errorf("container: no concrete found for: " + elem.String())
	}

	return fmt.Errorf("container: invalid abstraction")
}

// Fill takes a struct and fills the fields with the tag `container:"inject"`
func (c *Container) Fill(structure interface{}) error {
	receiverType := reflect.TypeOf(structure)
	if receiverType == nil {
		return fmt.Errorf("container: invalid structure")
	}

	if receiverType.Kind() == reflect.Ptr {
		elem := receiverType.Elem()
		if elem.Kind() == reflect.Struct {
			s := reflect.ValueOf(structure).Elem()
			for i := 0; i < s.NumField(); i++ {
				f := s.Field(i)

				if t, ok := s.Type().Field(i).Tag.Lookup(TagInject); ok {
					var concrete *binding
					var exist bool
					if len(t) > 0 {
						// e.g. $inject:"idGenerator"
						concrete, exist = c.idContainer[t]
					} else {
						// e.g. $inject:""
						concrete, exist = c.typeContainer[f.Type()]
					}

					if exist {
						instance, err := concrete.resolve(c)
						if err != nil {
							return err
						}

						ptr := reflect.NewAt(f.Type(), unsafe.Pointer(f.UnsafeAddr())).Elem()
						if ptr.CanSet() {
							ptr.Set(reflect.ValueOf(instance))
							continue
						}
						// try to use setter
						success := tryToGetSetterAndSet(ptr, f.Type().Name(), reflect.ValueOf(instance))
						if !success {
							return fmt.Errorf("container: cannot resolve %v field. please expose the field or make a setter for the field.", s.Type().Field(i).Name)
						}
						continue
					}
					return fmt.Errorf("container: cannot resolve %v field", s.Type().Field(i).Name)
				}
			}

			return nil
		}
	}

	return fmt.Errorf("container: invalid structure")
}

func tryToGetSetterAndSet(object reflect.Value, fieldName string, value interface{}) bool {
	// maybe try setXXX too?
	setterMethodName := fmt.Sprintf("%s%s", setterMethodPrefix, utils.ToPascalCase(fieldName))
	mv := object.MethodByName(setterMethodName)
	if !(mv.IsValid() && mv.Kind() == reflect.Func) {
		return false
	}
	mv.Call([]reflect.Value{reflect.ValueOf(value)})
	if recover() != nil {
		return false
	}
	return true
}
