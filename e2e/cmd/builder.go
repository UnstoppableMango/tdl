package main

type TestBuilder interface {
	Done() (Test, bool)
	WithName(string)
	WithSource(string)
	WithTarget(string)
}

type state struct {
	Name   *string
	Source *string
	Target *string
}

func NewTestBuilder() TestBuilder {
	return &state{}
}

// Done implements TestBuilder.
func (s state) Done() (Test, bool) {
	if s.Name == nil {
		return Test{}, false
	}
	if s.Source == nil {
		return Test{}, false
	}
	if s.Target == nil {
		return Test{}, false
	}

	return Test{
		Name:   *s.Name,
		Source: *s.Source,
		Target: *s.Target,
	}, true
}

// WithName implements TestBuilder.
func (s *state) WithName(name string) {
	s.Name = &name
}

// WithSource implements TestBuilder.
func (s *state) WithSource(source string) {
	s.Source = &source
}

// WithTarget implements TestBuilder.
func (s *state) WithTarget(target string) {
	s.Target = &target
}

var _ TestBuilder = &state{}
