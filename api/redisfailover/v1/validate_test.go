package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestValidateCustomCommandRenames(t *testing.T) {
	tests := []struct {
		name          string
		renames       []RedisCommandRename
		expectedError string
	}{
		{
			name:    "no renames",
			renames: nil,
		},
		{
			name:    "plain command rename",
			renames: []RedisCommandRename{{From: "CONFIG", To: "CONFIG2"}},
		},
		{
			name:    "renaming to an opaque secret",
			renames: []RedisCommandRename{{From: "FLUSHALL", To: "b840fc02d524045429941cc15f59e41cb7be6c52"}},
		},
		{
			name:    "empty replacement disables the command",
			renames: []RedisCommandRename{{From: "CONFIG", To: ""}},
		},
		{
			name:          "empty command name",
			renames:       []RedisCommandRename{{From: "", To: "CONFIG2"}},
			expectedError: `customCommandRenames: "" is not a valid command name`,
		},
		{
			// The proof of concept from the upstream report: the quote closes
			// the rename-command directive so the rest is parsed as config.
			name: "quote in the command name injects directives",
			renames: []RedisCommandRename{{
				From: "CONFIG\"\nslave-read-only no\nrename-command \"FLUSHALL",
				To:   `""`,
			}},
			expectedError: "is not a valid command name",
		},
		{
			name:          "quote in the replacement",
			renames:       []RedisCommandRename{{From: "CONFIG", To: `a" "b`}},
			expectedError: "is not a valid replacement for command",
		},
		{
			name:          "newline in the replacement",
			renames:       []RedisCommandRename{{From: "CONFIG", To: "a\nmaxmemory 1"}},
			expectedError: "is not a valid replacement for command",
		},
		{
			name:          "space in the replacement",
			renames:       []RedisCommandRename{{From: "CONFIG", To: "a b"}},
			expectedError: "is not a valid replacement for command",
		},
		{
			name:          "command name starting with a digit",
			renames:       []RedisCommandRename{{From: "1CONFIG", To: "x"}},
			expectedError: "is not a valid command name",
		},
		{
			name:          "later entry is validated too",
			renames:       []RedisCommandRename{{From: "CONFIG", To: "safe"}, {From: `BAD" "x`, To: "y"}},
			expectedError: "is not a valid command name",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			rf := &RedisFailover{
				ObjectMeta: metav1.ObjectMeta{Name: "test"},
				Spec: RedisFailoverSpec{
					Redis: RedisSettings{CustomCommandRenames: test.renames},
				},
			}

			err := rf.Validate()

			if test.expectedError == "" {
				assert.NoError(err)
			} else {
				assert.Error(err)
				assert.Contains(err.Error(), test.expectedError)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name                   string
		rfName                 string
		rfBootstrapNode        *BootstrapSettings
		rfRedisCustomConfig    []string
		rfSentinelCustomConfig []string
		expectedError          string
		expectedBootstrapNode  *BootstrapSettings
	}{
		{
			name:   "populates default values",
			rfName: "test",
		},
		{
			name:          "errors on too long of name",
			rfName:        "some-super-absurdely-unnecessarily-long-name-that-will-most-definitely-fail",
			expectedError: "name length can't be higher than 48",
		},
		{
			name:                   "SentinelCustomConfig provided",
			rfName:                 "test",
			rfSentinelCustomConfig: []string{"failover-timeout 500"},
		},
		{
			name:            "BootstrapNode provided without a host",
			rfName:          "test",
			rfBootstrapNode: &BootstrapSettings{},
			expectedError:   "BootstrapNode must include a host when provided",
		},
		{
			name:   "SentinelCustomConfig provided",
			rfName: "test",
		},
		{
			name:                  "Populates default bootstrap port when valid",
			rfName:                "test",
			rfBootstrapNode:       &BootstrapSettings{Host: "127.0.0.1"},
			expectedBootstrapNode: &BootstrapSettings{Host: "127.0.0.1", Port: "6379"},
		},
		{
			name:                  "Allows for specifying boostrap port",
			rfName:                "test",
			rfBootstrapNode:       &BootstrapSettings{Host: "127.0.0.1", Port: "6380"},
			expectedBootstrapNode: &BootstrapSettings{Host: "127.0.0.1", Port: "6380"},
		},
		{
			name:                "Appends applied custom config to default initial values",
			rfName:              "test",
			rfRedisCustomConfig: []string{"tcp-keepalive 60"},
		},
		{
			name:                  "Appends applied custom config to default initial values when bootstrapping",
			rfName:                "test",
			rfRedisCustomConfig:   []string{"tcp-keepalive 60"},
			rfBootstrapNode:       &BootstrapSettings{Host: "127.0.0.1"},
			expectedBootstrapNode: &BootstrapSettings{Host: "127.0.0.1", Port: "6379"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)
			rf := generateRedisFailover(test.rfName, test.rfBootstrapNode)
			rf.Spec.Redis.CustomConfig = test.rfRedisCustomConfig
			rf.Spec.Sentinel.CustomConfig = test.rfSentinelCustomConfig

			err := rf.Validate()

			if test.expectedError == "" {
				assert.NoError(err)

				expectedRedisCustomConfig := []string{
					"replica-priority 100",
				}

				if test.rfBootstrapNode != nil {
					expectedRedisCustomConfig = []string{
						"replica-priority 0",
					}
				}

				expectedRedisCustomConfig = append(expectedRedisCustomConfig, test.rfRedisCustomConfig...)
				expectedSentinelCustomConfig := defaultSentinelCustomConfig
				if len(test.rfSentinelCustomConfig) > 0 {
					expectedSentinelCustomConfig = test.rfSentinelCustomConfig
				}

				expectedRF := &RedisFailover{
					ObjectMeta: metav1.ObjectMeta{
						Name:      test.rfName,
						Namespace: "namespace",
					},
					Spec: RedisFailoverSpec{
						Redis: RedisSettings{
							Image:    defaultImage,
							Replicas: defaultRedisNumber,
							Port:     defaultRedisPort,
							Exporter: Exporter{
								Image: defaultExporterImage,
							},
							CustomConfig: expectedRedisCustomConfig,
						},
						Sentinel: SentinelSettings{
							Image:        defaultImage,
							Replicas:     defaultSentinelNumber,
							CustomConfig: expectedSentinelCustomConfig,
							Exporter: Exporter{
								Image: defaultSentinelExporterImage,
							},
						},
						BootstrapNode: test.expectedBootstrapNode,
					},
				}
				assert.Equal(expectedRF, rf)
			} else {
				if assert.Error(err) {
					assert.Contains(test.expectedError, err.Error())
				}
			}
		})
	}
}
