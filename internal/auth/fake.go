package auth

import "sync"

// Fake is an in-memory Keyring for tests. Safe for concurrent use.
type Fake struct {
	mu   sync.Mutex
	data map[string]string
}

// NewFake returns an empty in-memory keyring.
func NewFake() *Fake {
	return &Fake{data: map[string]string{}}
}

func (f *Fake) Set(ref, secret string) error {
	if ref == "" {
		return ErrEmptyRef
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.data[ref] = secret
	return nil
}

func (f *Fake) Get(ref string) (string, error) {
	if ref == "" {
		return "", ErrEmptyRef
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	secret, ok := f.data[ref]
	if !ok {
		return "", ErrNotFound
	}
	return secret, nil
}

func (f *Fake) Delete(ref string) error {
	if ref == "" {
		return ErrEmptyRef
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.data, ref)
	return nil
}

// Len returns the number of stored secrets. Useful for assertions.
func (f *Fake) Len() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.data)
}
