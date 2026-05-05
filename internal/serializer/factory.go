package serializer

import (
	"fmt"
	"sync"
)

// Factory manages serializer instances and provides mode-based selection.
type Factory struct {
	mu          sync.RWMutex
	mode        SerializerMode
	serializers map[SerializerMode]Serializer
}

// NewFactory creates a new serializer factory with the given default mode.
func NewFactory(mode SerializerMode) *Factory {
	f := &Factory{
		mode:        mode,
		serializers: make(map[SerializerMode]Serializer),
	}
	// Register default serializers
	f.serializers[ModeJSON] = NewJSONSerializer()
	f.serializers[ModeProtobuf] = NewProtobufSerializer()
	return f
}

// GetSerializer returns the current default serializer.
func (f *Factory) GetSerializer() Serializer {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.serializers[f.mode]
}

// GetSerializerByMode returns the serializer for the given mode.
// Returns an error if the mode is not registered.
func (f *Factory) GetSerializerByMode(mode SerializerMode) (Serializer, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	s, ok := f.serializers[mode]
	if !ok {
		return nil, fmt.Errorf("serializer: unknown mode %q", mode)
	}
	return s, nil
}

// SetMode sets the default serialization mode.
func (f *Factory) SetMode(mode SerializerMode) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.serializers[mode]; !ok {
		return fmt.Errorf("serializer: unknown mode %q", mode)
	}
	f.mode = mode
	return nil
}

// Register adds a custom serializer for the given mode.
func (f *Factory) Register(mode SerializerMode, s Serializer) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.serializers[mode] = s
}

// Mode returns the current default mode.
func (f *Factory) Mode() SerializerMode {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.mode
}
