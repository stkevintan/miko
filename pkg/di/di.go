package di

import (
	"context"
	"fmt"
	"reflect"
)

type containerKey struct{}

type container struct {
	services      map[reflect.Type]any
	namedServices map[string]any
}

// NewContext returns a new context with an empty DI container.
func NewContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, containerKey{}, &container{
		services:      make(map[reflect.Type]any),
		namedServices: make(map[string]any),
	})
}

// Provide registers a service in the container within the context.
func Provide[T any](ctx context.Context, service T) {
	c, ok := ctx.Value(containerKey{}).(*container)
	if !ok {
		panic("di: context does not contain a container")
	}
	c.services[reflect.TypeOf((*T)(nil)).Elem()] = service
}

// ProvideNamed registers a named service in the container within the context.
func ProvideNamed[T any](ctx context.Context, name string, service T) {
	c, ok := ctx.Value(containerKey{}).(*container)
	if !ok {
		panic("di: context does not contain a container")
	}
	c.namedServices[name] = service
}

// Invoke retrieves a service from the container within the context.
func Invoke[T any](ctx context.Context) (T, error) {
	c, ok := ctx.Value(containerKey{}).(*container)
	if !ok {
		var zero T
		return zero, fmt.Errorf("di: context does not contain a container")
	}
	s, ok := c.services[reflect.TypeOf((*T)(nil)).Elem()]
	if !ok {
		var zero T
		return zero, fmt.Errorf("di: service %T not found", zero)
	}
	return s.(T), nil
}

// InvokeNamed retrieves a named service from the container within the context.
func InvokeNamed[T any](ctx context.Context, name string) (T, error) {
	c, ok := ctx.Value(containerKey{}).(*container)
	if !ok {
		var zero T
		return zero, fmt.Errorf("di: context does not contain a container")
	}
	s, ok := c.namedServices[name]
	if !ok {
		var zero T
		return zero, fmt.Errorf("di: named service %s not found", name)
	}
	return s.(T), nil
}

// MustInvoke retrieves a service from the container or panics if not found.
func MustInvoke[T any](ctx context.Context) T {
	s, err := Invoke[T](ctx)
	if err != nil {
		panic(err)
	}
	return s
}
