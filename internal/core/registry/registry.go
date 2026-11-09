package registry

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"i3_next/internal/api/contracts"
)

// Service is the core service contract.
type Service interface {
	Name() string
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Health(ctx context.Context) contracts.ServiceInfo
}

// Reloader is an optional capability implemented by services that support
// reloading their configuration at runtime.
type Reloader interface {
	Reload(ctx context.Context) error
}

// Requirement classifies a service as required or optional.
type Requirement string

const (
	// Required marks a service whose failure is fatal.
	Required Requirement = "required"
	// Optional marks a service whose failure degrades the system.
	Optional Requirement = "optional"
)

// Entry describes a registered service.
type Entry struct {
	Name        string
	Service     Service
	Required    bool
	Optional    bool
	Requirement Requirement
}

// Registry holds registered services.
type Registry struct {
	mu      sync.RWMutex
	order   []string
	entries map[string]Entry
}

// New creates a new Registry.
func New() *Registry {
	return &Registry{entries: make(map[string]Entry)}
}

// DefaultRegistry is a package-level registry.
var DefaultRegistry = New()

// marker interfaces (unexported) for classifying services.
type requiredMarker interface{ Required() bool }
type optionalMarker interface{ Optional() bool }
type requirementProvider interface{ Requirement() Requirement }

// RequiredService wraps a Service as required.
type RequiredService struct {
	Service Service
}

func (RequiredService) Required() bool { return true }

func (r RequiredService) Name() string {
	if r.Service != nil {
		return r.Service.Name()
	}
	return ""
}
func (r RequiredService) Start(ctx context.Context) error {
	if r.Service != nil {
		return r.Service.Start(ctx)
	}
	return nil
}
func (r RequiredService) Stop(ctx context.Context) error {
	if r.Service != nil {
		return r.Service.Stop(ctx)
	}
	return nil
}
func (r RequiredService) Health(ctx context.Context) contracts.ServiceInfo {
	if r.Service != nil {
		return r.Service.Health(ctx)
	}
	return contracts.ServiceInfo{}
}

// OptionalService wraps a Service as optional.
type OptionalService struct {
	Service Service
}

func (OptionalService) Optional() bool { return true }

func (o OptionalService) Name() string {
	if o.Service != nil {
		return o.Service.Name()
	}
	return ""
}
func (o OptionalService) Start(ctx context.Context) error {
	if o.Service != nil {
		return o.Service.Start(ctx)
	}
	return nil
}
func (o OptionalService) Stop(ctx context.Context) error {
	if o.Service != nil {
		return o.Service.Stop(ctx)
	}
	return nil
}
func (o OptionalService) Health(ctx context.Context) contracts.ServiceInfo {
	if o.Service != nil {
		return o.Service.Health(ctx)
	}
	return contracts.ServiceInfo{}
}

// WrapRequired wraps a service as required.
func WrapRequired(s Service) Service { return RequiredService{Service: s} }

// WrapOptional wraps a service as optional.
func WrapOptional(s Service) Service { return OptionalService{Service: s} }

// RegisterOption configures a single registration.
type RegisterOption func(*registerOptions)

type registerOptions struct {
	name   string
	req    Requirement
	hasReq bool
}

// Name assigns the service a registration name (defaults to Service.Name()).
func Name(n string) RegisterOption {
	return func(o *registerOptions) { o.name = n }
}

// WithRequirement sets the service's required/optional classification.
func WithRequirement(r Requirement) RegisterOption {
	return func(o *registerOptions) { o.req = r; o.hasReq = true }
}

// Register adds a service to the registry.
func (r *Registry) Register(s Service, opts ...RegisterOption) error {
	if r == nil {
		return errors.New("nil registry")
	}
	if s == nil {
		return fmt.Errorf("%w: nil service", ErrInvalidArgument)
	}
	o := registerOptions{req: Required}
	for _, opt := range opts {
		if opt != nil {
			opt(&o)
		}
	}
	name := o.name
	if name == "" {
		name = s.Name()
	}
	if name == "" {
		return fmt.Errorf("%w: empty service name", ErrInvalidArgument)
	}
	required := o.req == Required
	optional := !required
	req := o.req

	if !o.hasReq {
		// Infer classification from the service itself.
		if rm, ok := s.(requiredMarker); ok && rm.Required() {
			required, optional, req = true, false, Required
		} else if om, ok := s.(optionalMarker); ok && om.Optional() {
			required, optional, req = false, true, Optional
		} else if rp, ok := s.(requirementProvider); ok {
			req = rp.Requirement()
			required = req == Required
			optional = req == Optional
		}
	}
	if required {
		optional = false
	}
	if optional {
		required = false
	}
	if req == "" {
		if required {
			req = Required
		} else {
			req = Optional
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.entries == nil {
		r.entries = make(map[string]Entry)
	}
	if _, exists := r.entries[name]; exists {
		return fmt.Errorf("%w: %s", ErrAlreadyRegistered, name)
	}
	entry := Entry{
		Name:        name,
		Service:     s,
		Required:    required,
		Optional:    optional,
		Requirement: req,
	}
	r.entries[name] = entry
	r.order = append(r.order, name)
	return nil
}

// RegisterRequired registers a required service.
func (r *Registry) RegisterRequired(s Service) error {
	return r.Register(s, WithRequirement(Required))
}

// RegisterOptional registers an optional service.
func (r *Registry) RegisterOptional(s Service) error {
	return r.Register(s, WithRequirement(Optional))
}

// MustRegister registers or panics.
func (r *Registry) MustRegister(s Service, opts ...RegisterOption) {
	if err := r.Register(s, opts...); err != nil {
		panic(err)
	}
}

// Get returns a service by name.
func (r *Registry) Get(name string) (Service, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	e, ok := r.entries[name]
	if !ok {
		return nil, false
	}
	return e.Service, true
}

// Entry returns the full entry.
func (r *Registry) Entry(name string) (Entry, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	e, ok := r.entries[name]
	return e, ok
}

// Services returns all services in registration order.
func (r *Registry) Services() []Service {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Service, 0, len(r.order))
	for _, n := range r.order {
		if e, ok := r.entries[n]; ok {
			out = append(out, e.Service)
		}
	}
	return out
}

// Entries returns all entries.
func (r *Registry) Entries() []Entry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Entry, 0, len(r.order))
	for _, n := range r.order {
		if e, ok := r.entries[n]; ok {
			out = append(out, e)
		}
	}
	return out
}

// Required returns required services.
func (r *Registry) Required() []Service {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Service, 0)
	for _, n := range r.order {
		if e, ok := r.entries[n]; ok && e.Required {
			out = append(out, e.Service)
		}
	}
	return out
}

// Optional returns optional services.
func (r *Registry) Optional() []Service {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Service, 0)
	for _, n := range r.order {
		if e, ok := r.entries[n]; ok && e.Optional {
			out = append(out, e.Service)
		}
	}
	return out
}

// IsRequired reports whether a named service is required.
func (r *Registry) IsRequired(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	e, ok := r.entries[name]
	return ok && e.Required
}

// IsOptional reports whether a named service is optional.
func (r *Registry) IsOptional(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	e, ok := r.entries[name]
	return ok && e.Optional
}

// RequirementOf returns the requirement of a named service.
func (r *Registry) RequirementOf(name string) (Requirement, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	e, ok := r.entries[name]
	if !ok {
		return "", false
	}
	return e.Requirement, true
}

// Has reports whether a service is registered.
func (r *Registry) Has(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.entries[name]
	return ok
}

// Len returns the number of registered services.
func (r *Registry) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.entries)
}

// Unregister removes a service.
func (r *Registry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.entries[name]; !ok {
		return fmt.Errorf("%w: %s", ErrNotFound, name)
	}
	delete(r.entries, name)
	for i, n := range r.order {
		if n == name {
			r.order = append(r.order[:i], r.order[i+1:]...)
			break
		}
	}
	return nil
}

// Clear removes all services.
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries = make(map[string]Entry)
	r.order = nil
}

// Range iterates over services.
func (r *Registry) Range(fn func(name string, s Service, required bool) bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, n := range r.order {
		e, ok := r.entries[n]
		if !ok {
			continue
		}
		if !fn(e.Name, e.Service, e.Required) {
			break
		}
	}
}

// Names returns registered names.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, len(r.order))
	copy(out, r.order)
	return out
}

// IsRequiredService reports whether a service is required based on markers/wrappers.
func IsRequiredService(s Service) bool {
	if s == nil {
		return true
	}
	if rs, ok := s.(requiredMarker); ok {
		return rs.Required()
	}
	if om, ok := s.(optionalMarker); ok {
		return !om.Optional()
	}
	if rp, ok := s.(requirementProvider); ok {
		return rp.Requirement() == Required
	}
	switch s.(type) {
	case RequiredService, *RequiredService:
		return true
	case OptionalService, *OptionalService:
		return false
	}
	return true
}

// IsOptionalService reports whether a service is optional.
func IsOptionalService(s Service) bool { return !IsRequiredService(s) }

// errors
var (
	ErrInvalidArgument   = errors.New("invalid argument")
	ErrNotFound          = errors.New("not found")
	ErrAlreadyRegistered = errors.New("already registered")
)
