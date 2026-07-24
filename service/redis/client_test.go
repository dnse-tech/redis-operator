package redis

import (
	"errors"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsUnreachableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil is not unreachable",
			err:      nil,
			expected: false,
		},
		{
			name:     "i/o timeout string",
			err:      errors.New("dial tcp 10.0.0.1:6379: i/o timeout"),
			expected: true,
		},
		{
			name:     "connection refused string",
			err:      errors.New("dial tcp 10.0.0.1:6379: connect: connection refused"),
			expected: true,
		},
		{
			name:     "no route to host string",
			err:      errors.New("dial tcp 10.0.0.1:6379: connect: no route to host"),
			expected: true,
		},
		{
			name:     "network is unreachable string",
			err:      errors.New("dial tcp 10.0.0.1:6379: connect: network is unreachable"),
			expected: true,
		},
		{
			name:     "connection reset string",
			err:      errors.New("read tcp 10.0.0.1:6379: connection reset by peer"),
			expected: true,
		},
		{
			name:     "real net.Error is unreachable",
			err:      &net.DNSError{Err: "timeout", Name: "redis", IsTimeout: true},
			expected: true,
		},
		{
			name:     "net.OpError is unreachable",
			err:      &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connect: host is down")},
			expected: true,
		},
		{
			name:     "redis rejecting the command is not unreachable",
			err:      errors.New("ERR unknown parameter 'foo'"),
			expected: false,
		},
		{
			name:     "auth failure is not unreachable",
			err:      errors.New("NOAUTH Authentication required"),
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, IsUnreachableError(test.err))
		})
	}
}
