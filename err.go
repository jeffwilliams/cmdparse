package cmdparse

import "strings"

type Errors []error

func newErrors() Errors {
	return Errors([]error{})
}

func (e Errors) Error() string {
	if e == nil {
		return ""
	}

	s := []string{}
	for _, err := range e {
		s = append(s, err.Error())
	}
	return strings.Join(s, "\n")
}

func (e *Errors) add(err error) {
	*e = append(*e, err)
}

func (e Errors) nilIfEmpty() error {
	if len(e) == 0 {
		return nil
	}
	return e
}
