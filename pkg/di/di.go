package di

import (
	"context"
	"fmt"
	"reflect"
	"sync"
)

type containerKey struct{}

type factory[T any] struct {
	fn   func(context.Context) T
	once sync.Once
	val  T
}

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

// ProvideFactory registers a factory function that creates a service lazily on first access.
// The factory is called only once per container, and the result is cached.
func ProvideFactory[T any](ctx context.Context, fn func(context.Context) T) {
	c, ok := ctx.Value(containerKey{}).(*container)
	if !ok {
		panic("di: context does not contain a container")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.services[reflect.TypeOf((*T)(nil)).Elem()] = &factory[T]{fn: fn}
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
			// Check if it's a factory
			if f, isFactory := s.(*factory[T]); isFactory {
				f.once.Do(func() {
					f.val = f.fn(ctx)
				})
				return f.val, nil
			}
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

func MustInvokeNamed[T any](ctx context.Context, name string) T {
	s, err := InvokeNamed[T](ctx, name)
	if err != nil {
		panic(err)
	}
	return s
}
