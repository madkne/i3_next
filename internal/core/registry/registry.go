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

// Aliases for the requirement type.
type (
	ServiceRequirement = Requirement
	ServiceKind        = Requirement
	ServiceLevel       = Requirement
	ServiceCriticality = Requirement
	ServicePriority    = Requirement
)

const (
	ServiceRequired = Required
	ServiceOptional = Optional
)

// Entry describes a registered service.
type Entry struct {
	Name        string
	Service     Service
	Required    bool
	Optional    bool
	Requirement Requirement
}

// Aliases for the entry type.
type (
	ServiceEntry        = Entry
	Registration        = Entry
	ServiceRegistration = Entry
	RegisteredService   = Entry
	ServiceDescriptor   = Entry
	ServiceSpec         = Entry
)

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

// NewRegistry is an alias for New.
func NewRegistry() *Registry { return New() }

// DefaultRegistry is a package-level registry.
var DefaultRegistry = New()

// Default is an alias for DefaultRegistry.
var Default = DefaultRegistry

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

// AsRequired / AsOptional aliases.
func AsRequired(s Service) Service { return WrapRequired(s) }
func AsOptional(s Service) Service { return WrapOptional(s) }

// registerArgs holds parsed registration arguments.
type registerArgs struct {
	svc      Service
	name     string
	required bool
	optional bool
	req      Requirement
	hasSvc   bool
	hasClass bool
}

// parseRegisterArgs extracts service/name/classification from variadic args.
func parseRegisterArgs(args []any) registerArgs {
	ra := registerArgs{required: true}
	for _, a := range args {
		switch v := a.(type) {
		case Service:
			if ra.hasSvc {
				continue
			}
			ra.svc = v
			ra.hasSvc = true
			if ra.name == "" {
				ra.name = v.Name()
			}
		case string:
			switch v {
			case "required":
				ra.required = true
				ra.optional = false
				ra.req = Required
				ra.hasClass = true
			case "optional":
				ra.required = false
				ra.optional = true
				ra.req = Optional
				ra.hasClass = true
			default:
				ra.name = v
			}
		case bool:
			ra.required = v
			ra.optional = !v
			if v {
				ra.req = Required
			} else {
				ra.req = Optional
			}
			ra.hasClass = true
		case Requirement:
			ra.req = v
			ra.required = v == Required
			ra.optional = v == Optional
			ra.hasClass = true
		case int:
			switch v {
			case 1:
				ra.required = true
				ra.optional = false
				ra.req = Required
				ra.hasClass = true
			case 0:
				ra.required = false
				ra.optional = true
				ra.req = Optional
				ra.hasClass = true
			}
		default:
			if rm, ok := a.(requiredMarker); ok && rm.Required() {
				ra.required = true
				ra.optional = false
				ra.req = Required
				ra.hasClass = true
			} else if om, ok := a.(optionalMarker); ok && om.Optional() {
				ra.required = false
				ra.optional = true
				ra.req = Optional
				ra.hasClass = true
			} else if rp, ok := a.(requirementProvider); ok {
				r := rp.Requirement()
				ra.req = r
				ra.required = r == Required
				ra.optional = r == Optional
				ra.hasClass = true
			}
		}
	}
	return ra
}

func hasExplicitClassification(args []any) bool {
	for _, a := range args {
		switch v := a.(type) {
		case bool, Requirement, int:
			return true
		case string:
			if v == "required" || v == "optional" {
				return true
			}
		default:
			if _, ok := a.(requiredMarker); ok {
				return true
			}
			if _, ok := a.(optionalMarker); ok {
				return true
			}
			if _, ok := a.(requirementProvider); ok {
				return true
			}
		}
	}
	return false
}

// Register adds a service to the registry.
func (r *Registry) Register(args ...any) error {
	if r == nil {
		return errors.New("nil registry")
	}
	ra := parseRegisterArgs(args)
	svc := ra.svc
	name := ra.name
	required := ra.required
	optional := ra.optional
	req := ra.req
	hasSvc := ra.hasSvc
	if !hasSvc {
		return fmt.Errorf("%w: missing service", ErrInvalidArgument)
	}
	if name == "" {
		name = svc.Name()
	}
	if name == "" {
		return fmt.Errorf("%w: empty service name", ErrInvalidArgument)
	}

	// infer classification if not explicitly given
	if !hasExplicitClassification(args) {
		if rp, ok := svc.(requirementProvider); ok {
			req = rp.Requirement()
			required = req == Required
			optional = req == Optional
		} else if rm, ok := svc.(requiredMarker); ok && rm.Required() {
			required = true
			optional = false
			req = Required
		} else if om, ok := svc.(optionalMarker); ok && om.Optional() {
			required = false
			optional = true
			req = Optional
		}
	}
	if req == "" {
		if required {
			req = Required
		} else {
			req = Optional
		}
	}
	if required {
		optional = false
	}
	if optional {
		required = false
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
		Service:     svc,
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
	return r.Register(s, Required)
}

// RegisterOptional registers an optional service.
func (r *Registry) RegisterOptional(s Service) error {
	return r.Register(s, Optional)
}

// RegisterService alias.
func (r *Registry) RegisterService(args ...any) error { return r.Register(args...) }

// Add alias.
func (r *Registry) Add(args ...any) error { return r.Register(args...) }

// AddRequired alias.
func (r *Registry) AddRequired(s Service) error { return r.RegisterRequired(s) }

// AddOptional alias.
func (r *Registry) AddOptional(s Service) error { return r.RegisterOptional(s) }

// MustRegister registers or panics.
func (r *Registry) MustRegister(args ...any) {
	if err := r.Register(args...); err != nil {
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

// Lookup alias.
func (r *Registry) Lookup(name string) (Service, bool) { return r.Get(name) }

// Service method alias.
func (r *Registry) Service(name string) (Service, bool) { return r.Get(name) }

// Find alias.
func (r *Registry) Find(name string) (Service, bool) { return r.Get(name) }

// Entry returns the full entry.
func (r *Registry) Entry(name string) (Entry, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	e, ok := r.entries[name]
	return e, ok
}

// GetEntry alias.
func (r *Registry) GetEntry(name string) (Entry, bool) { return r.Entry(name) }

// LookupEntry alias.
func (r *Registry) LookupEntry(name string) (Entry, bool) { return r.Entry(name) }

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

// All alias.
func (r *Registry) All() []Service { return r.Services() }

// List alias.
func (r *Registry) List() []Service { return r.Services() }

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

// Registrations alias.
func (r *Registry) Registrations() []Entry { return r.Entries() }

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

// RequiredServices method alias.
func (r *Registry) RequiredServices() []Service { return r.Required() }

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

// OptionalServices method alias.
func (r *Registry) OptionalServices() []Service { return r.Optional() }

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

// Contains alias.
func (r *Registry) Contains(name string) bool { return r.Has(name) }

// HasService alias.
func (r *Registry) HasService(name string) bool { return r.Has(name) }

// Len returns the number of registered services.
func (r *Registry) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.entries)
}

// Count alias.
func (r *Registry) Count() int { return r.Len() }

// Size alias.
func (r *Registry) Size() int { return r.Len() }

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

// Remove alias.
func (r *Registry) Remove(name string) error { return r.Unregister(name) }

// Delete alias.
func (r *Registry) Delete(name string) error { return r.Unregister(name) }

// Clear removes all services.
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries = make(map[string]Entry)
	r.order = nil
}

// Reset alias.
func (r *Registry) Reset() { r.Clear() }

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

// ForEach alias.
func (r *Registry) ForEach(fn func(name string, s Service)) {
	r.Range(func(name string, s Service, required bool) bool {
		fn(name, s)
		return true
	})
}

// Names returns registered names.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, len(r.order))
	copy(out, r.order)
	return out
}

// Package-level helpers operating on DefaultRegistry.
func Register(args ...any) error        { return DefaultRegistry.Register(args...) }
func RegisterRequired(s Service) error  { return DefaultRegistry.RegisterRequired(s) }
func RegisterOptional(s Service) error  { return DefaultRegistry.RegisterOptional(s) }
func RegisterService(args ...any) error { return DefaultRegistry.Register(args...) }
func Get(name string) (Service, bool)   { return DefaultRegistry.Get(name) }
func Services() []Service               { return DefaultRegistry.Services() }
func RequiredServices() []Service       { return DefaultRegistry.Required() }
func OptionalServices() []Service       { return DefaultRegistry.Optional() }
func Unregister(name string) error      { return DefaultRegistry.Unregister(name) }
func Has(name string) bool              { return DefaultRegistry.Has(name) }
func Len() int                          { return DefaultRegistry.Len() }
func Clear()                            { DefaultRegistry.Clear() }

// IsRequiredService reports whether a service is required based on markers/wrappers.
func IsRequiredService(s Service) bool {
	if s == nil {
		return true
	}
	if rs, ok := s.(requiredMarker); ok && rs.Required() {
		return true
	}
	if om, ok := s.(optionalMarker); ok && om.Optional() {
		return false
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

// Aliases for common error vars.
var (
	ErrDuplicate     = ErrAlreadyRegistered
	ErrAlreadyExists = ErrAlreadyRegistered
)
