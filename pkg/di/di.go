package di

import (
	"context"
	"fmt"
	"reflect"
	"sync"
)

type containerKey struct{}

type container struct {
	mu            sync.RWMutex
	parent        *container
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

// Inherit creates a new child context from ctx that inherits services from the container in parent.
// If parent is not provided, it inherits from the container in ctx.
func Inherit(ctx context.Context, parent ...context.Context) context.Context {
	pCtx := ctx
	if len(parent) > 0 {
		pCtx = parent[0]
	}

	p, ok := pCtx.Value(containerKey{}).(*container)
	if !ok {
		return NewContext(ctx)
	}

	child := &container{
		parent:        p,
		services:      make(map[reflect.Type]any),
		namedServices: make(map[string]any),
	}

	return context.WithValue(ctx, containerKey{}, child)
}

// Provide registers a service in the container within the context.
func Provide[T any](ctx context.Context, service T) {
	c, ok := ctx.Value(containerKey{}).(*container)
	if !ok {
		panic("di: context does not contain a container")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.services[reflect.TypeOf((*T)(nil)).Elem()] = service
}

// ProvideNamed registers a named service in the container within the context.
func ProvideNamed[T any](ctx context.Context, name string, service T) {
	c, ok := ctx.Value(containerKey{}).(*container)
	if !ok {
		panic("di: context does not contain a container")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.namedServices[name] = service
}

// Invoke retrieves a service from the container within the context.
func Invoke[T any](ctx context.Context) (T, error) {
	c, ok := ctx.Value(containerKey{}).(*container)
	if !ok {
		var zero T
		return zero, fmt.Errorf("di: context does not contain a container")
	}

	targetType := reflect.TypeOf((*T)(nil)).Elem()

	// Search current container and parent chain
	for c != nil {
		c.mu.RLock()
		s, ok := c.services[targetType]
		c.mu.RUnlock()

		if ok {
			return s.(T), nil
		}

		c = c.parent
	}

	var zero T
	return zero, fmt.Errorf("di: service %T not found", zero)
}

// InvokeNamed retrieves a named service from the container within the context.
func InvokeNamed[T any](ctx context.Context, name string) (T, error) {
	c, ok := ctx.Value(containerKey{}).(*container)
	if !ok {
		var zero T
		return zero, fmt.Errorf("di: context does not contain a container")
	}

	// Search current container and parent chain
	for c != nil {
		c.mu.RLock()
		s, ok := c.namedServices[name]
		c.mu.RUnlock()

		if ok {
			return s.(T), nil
		}

		c = c.parent
	}

	var zero T
	return zero, fmt.Errorf("di: named service %s not found", name)
}

// MustInvoke retrieves a service from the container or panics if not found.
func MustInvoke[T any](ctx context.Context) T {
	s, err := Invoke[T](ctx)
	if err != nil {
		panic(err)
	}
	return s
}
