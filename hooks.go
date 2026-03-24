package s3mock

// OnPut registers a callback that fires after every successful PutObject.
// Multiple hooks can be registered and all will fire in order.
func (m *Mock) OnPut(fn func(bucket, key string)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onPut = append(m.onPut, fn)
}

// OnDelete registers a callback that fires after every successful DeleteObject.
// Multiple hooks can be registered and all will fire in order.
func (m *Mock) OnDelete(fn func(bucket, key string)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onDelete = append(m.onDelete, fn)
}
