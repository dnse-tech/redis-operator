package k8s

import (
	apiextensionscli "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"

	redisfailoverclientset "github.com/dnse-tech/redis-operator/client/k8s/clientset/versioned"
	"github.com/dnse-tech/redis-operator/log"
	"github.com/dnse-tech/redis-operator/metrics"
)

// Service is the K8s service entrypoint.
type Services interface {
	ConfigMap
	Secret
	Pod
	PodDisruptionBudget
	RedisFailover
	Service
	RBAC
	Deployment
	StatefulSet
}

type services struct {
	ConfigMap
	Secret
	Pod
	PodDisruptionBudget
	RedisFailover
	Service
	RBAC
	Deployment
	StatefulSet
}

// New returns a new Kubernetes service.
func New(kubecli kubernetes.Interface, crdcli redisfailoverclientset.Interface, apiextcli apiextensionscli.Interface, logger log.Logger, metricsRecorder metrics.Recorder, opts ...ServiceOption) Services {
	return &services{
		ConfigMap:           NewConfigMapService(kubecli, logger, metricsRecorder, opts...),
		Secret:              NewSecretService(kubecli, logger, metricsRecorder),
		Pod:                 NewPodService(kubecli, logger, metricsRecorder),
		PodDisruptionBudget: NewPodDisruptionBudgetService(kubecli, logger, metricsRecorder, opts...),
		RedisFailover:       NewRedisFailoverService(crdcli, logger, metricsRecorder),
		Service:             NewServiceService(kubecli, logger, metricsRecorder, opts...),
		RBAC:                NewRBACService(kubecli, logger, metricsRecorder, opts...),
		Deployment:          NewDeploymentService(kubecli, logger, metricsRecorder, opts...),
		StatefulSet:         NewStatefulSetService(kubecli, logger, metricsRecorder, opts...),
	}
}
