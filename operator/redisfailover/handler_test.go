package redisfailover_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	redisfailoverv1 "github.com/dnse-tech/redis-operator/api/redisfailover/v1"
	"github.com/dnse-tech/redis-operator/log"
	"github.com/dnse-tech/redis-operator/metrics"
	rfOperator "github.com/dnse-tech/redis-operator/operator/redisfailover"
)

// The skip-reconcile check sits ahead of validation, so a RedisFailover with a
// deliberately invalid name tells the two paths apart without wiring up every
// downstream service: skipping returns nil, reconciling reaches validation and
// fails. Either way nothing below the gate is reached, so nil collaborators are
// safe here.
func TestHandleSkipReconcileAnnotation(t *testing.T) {
	tests := []struct {
		name        string
		annotations map[string]string
		skipped     bool
	}{
		{
			name:        "no annotations at all",
			annotations: nil,
			skipped:     false,
		},
		{
			name:        "unrelated annotation",
			annotations: map[string]string{"example.com/other": "true"},
			skipped:     false,
		},
		{
			name:        "skip-reconcile true",
			annotations: map[string]string{"skip-reconcile": "true"},
			skipped:     true,
		},
		{
			name:        "skip-reconcile accepts other truthy spellings",
			annotations: map[string]string{"skip-reconcile": "True"},
			skipped:     true,
		},
		{
			name:        "skip-reconcile false still reconciles",
			annotations: map[string]string{"skip-reconcile": "false"},
			skipped:     false,
		},
		{
			name:        "unparseable value falls back to reconciling",
			annotations: map[string]string{"skip-reconcile": "maybe"},
			skipped:     false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			rf := &redisfailoverv1.RedisFailover{
				ObjectMeta: metav1.ObjectMeta{
					// Longer than the 48 character limit, so validation fails.
					Name:        strings.Repeat("a", 60),
					Namespace:   "testns",
					Annotations: test.annotations,
				},
			}

			handler := rfOperator.NewRedisFailoverHandler(
				rfOperator.Config{}, nil, nil, nil, nil, metrics.Dummy, log.Dummy,
			)

			err := handler.Handle(context.Background(), rf)

			if test.skipped {
				assert.NoError(err, "expected reconcile to be skipped before validation")
			} else {
				assert.Error(err, "expected reconcile to proceed and hit validation")
			}
		})
	}
}
