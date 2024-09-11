package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
)

var (
	ErrRpcCallNotFound = errors.New("rpc call not found")
)

type MemoryAdapter struct {
	mtx sync.RWMutex
	m   map[string]func(ctx context.Context, in json.RawMessage) (out any, err error)
}

// var _ rpcAdaptor = (*MemoryAdapter)(nil)

func (m *MemoryAdapter) Request(ctx context.Context, topic string, in any) (json.RawMessage, error) {
	slog.DebugContext(ctx, "rpc call", "topic", topic)

	m.mtx.RLock()
	fn, ok := m.m[topic]
	m.mtx.RUnlock()

	if !ok {
		return nil, ErrRpcCallNotFound
	}

	b, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}

	result, err := fn(ctx, b)
	if err != nil {
		return nil, err
	}

	b, err = json.Marshal(result)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func (m *MemoryAdapter) Reply(ctx context.Context, topic string, fn func(ctx context.Context, in json.RawMessage) (out any, err error)) error {
	slog.DebugContext(ctx, "register rpc function", "topic", topic)

	m.mtx.Lock()
	m.m[topic] = fn
	m.mtx.Unlock()

	return nil
}

func NewMemory() *MemoryAdapter {
	return &MemoryAdapter{
		m: make(map[string]func(ctx context.Context, in json.RawMessage) (out any, err error)),
	}
}
