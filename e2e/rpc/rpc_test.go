package rpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRpcCall(t *testing.T) {
	mem := NewMemoryHandleRegistry()

	RegisterRpcGreetingServiceServer(mem, &RpcGreetingServiceImpl{})

	caller := NewRpcCallerMemory(mem)

	client := CreateRpcGreetingServiceClient(caller)

	resp, err := client.SayHello(context.Background(), "World")
	assert.NoError(t, err)
	assert.Equal(t, "Hello World", resp)
}
