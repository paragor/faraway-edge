package k8s

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/paragor/faraway-edge/pkg/encodinghelper"
	"github.com/paragor/faraway-edge/pkg/envoy"
	"github.com/paragor/faraway-edge/pkg/log"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

type IngressProvider struct {
	clientset kubernetes.Interface
	informer  cache.SharedIndexInformer
	queue     workqueue.TypedRateLimitingInterface[string]

	mu      sync.RWMutex
	cluster *envoy.LogicalCluster

	ingressClasses []string
	clusterName    string
}

func NewIngressProvider(
	clusterName string,
	ingressClasses []string,
	clientset kubernetes.Interface,
	resyncPeriod time.Duration,
) (*IngressProvider, error) {
	informerFactory := informers.NewSharedInformerFactory(clientset, resyncPeriod)
	informer := informerFactory.Networking().V1().Ingresses().Informer()

	p := &IngressProvider{
		clusterName:    clusterName,
		clientset:      clientset,
		ingressClasses: ingressClasses,
		informer:       informer,
		queue:          workqueue.NewTypedRateLimitingQueue[string](workqueue.DefaultTypedControllerRateLimiter[string]()),
	}

	handler := func() {
		p.queue.Add("reconcile")
	}

	_, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			handler()
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			handler()
		},
		DeleteFunc: func(obj interface{}) {
			handler()
		},
	})
	if err != nil {
		return nil, fmt.Errorf("error adding ingress informer: %w", err)
	}
	return p, nil
}

func (p *IngressProvider) Run(ctx context.Context) error {
	defer p.queue.ShutDown()

	logger := log.FromContext(ctx)
	logger.Info("starting ingress provider")

	go p.informer.Run(ctx.Done())

	if !cache.WaitForCacheSync(ctx.Done(), p.informer.HasSynced) {
		return ctx.Err()
	}

	logger.Info("informer cache synced, starting worker")

	go wait.UntilWithContext(ctx, p.worker, time.Second)

	<-ctx.Done()
	logger.Info("shutting down ingress provider")
	return nil
}
func (p *IngressProvider) worker(ctx context.Context) {
	for p.processNextWorkItem(ctx) {
	}
}
func (p *IngressProvider) processNextWorkItem(ctx context.Context) bool {
	key, quit := p.queue.Get()
	if quit {
		return false
	}
	defer p.queue.Done(key)

	err := p.reconcile(ctx)
	if err != nil {
		log.FromContext(ctx).Error("reconciliation failed", log.Error(err))

		p.queue.AddRateLimited(key)
		return true
	}

	p.queue.Forget(key)
	return true
}

func (p *IngressProvider) reconcile(ctx context.Context) error {
	objs := p.informer.GetStore().List()

	ingresses := make([]*networkingv1.Ingress, 0, len(objs))
	for _, obj := range objs {
		ingress := obj.(*networkingv1.Ingress)
		ingresses = append(ingresses, ingress)
	}

	newCluster := p.covertIngressToLogicaCluster(ctx, ingresses)

	p.mu.Lock()
	p.cluster = newCluster
	p.mu.Unlock()

	return nil
}

func (p *IngressProvider) GetLogicaCluster(ctx context.Context) (*envoy.LogicalCluster, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.cluster == nil {
		return nil, fmt.Errorf("not ready")
	}

	return p.cluster, nil
}

func (p *IngressProvider) covertIngressToLogicaCluster(ctx context.Context, ingresses []*networkingv1.Ingress) *envoy.LogicalCluster {
	ingresses = slices.DeleteFunc(ingresses, func(ingress *networkingv1.Ingress) bool {
		annotations := ingress.GetAnnotations()
		if annotations[annotationEnabled] != "true" {
			return true
		}

		if len(p.ingressClasses) > 0 {
			ingressClassName := annotations["kubernetes.io/ingress.class"]
			if ingress.Spec.IngressClassName != nil {
				ingressClassName = *ingress.Spec.IngressClassName
			}

			if !slices.Contains(p.ingressClasses, ingressClassName) {
				return true
			}
		}

		if len(ingress.Status.LoadBalancer.Ingress) == 0 {
			return true
		}

		if len(p.collectHosts(ingress)) == 0 {
			return true
		}
		if len(p.collectBalancerIps(ingress)) == 0 {
			return true
		}
		return false
	})
	view := &envoy.LogicalCluster{
		Name: p.clusterName,
	}
	for _, ingress := range ingresses {
		logicalIngress := &envoy.LogicalClusterIngress{
			Name: ingress.GetNamespace() + "/" + ingress.GetName(),
		}
		ips := p.collectBalancerIps(ingress)
		hosts := p.collectHosts(ingress)
		timeout := p.getConnectionTimeout(ctx, ingress)

		logicalIngress.HttpsUpstream = &envoy.EnvoyUpstreamStaticAddresses{
			Port:            443,
			StaticAddresses: ips,
			ConnectTimeout:  encodinghelper.NewDuration(timeout),
		}
		logicalIngress.HttpUpstream = &envoy.EnvoyUpstreamStaticAddresses{
			Port:            80,
			StaticAddresses: ips,
			ConnectTimeout:  encodinghelper.NewDuration(timeout),
		}
		for _, host := range hosts {
			logicalIngress.Frontends = append(logicalIngress.Frontends, &envoy.IngressConfig{
				Domain: host,
			})
		}
		view.Ingresses = append(view.Ingresses, logicalIngress)
	}

	return view

}

func (p *IngressProvider) collectBalancerIps(ingress *networkingv1.Ingress) []string {
	ips := []string{}
	for _, status := range ingress.Status.LoadBalancer.Ingress {
		if status.IP != "" {
			ips = append(ips, status.IP)
		}
	}
	return ips
}

func (p *IngressProvider) collectHosts(ingress *networkingv1.Ingress) []string {
	hosts := []string{}
	for _, rule := range ingress.Spec.Rules {
		if rule.Host == "" {
			continue
		}
		hosts = append(hosts, rule.Host)
	}
	annotationWithHost := ingress.GetAnnotations()["nginx.ingress.kubernetes.io/server-alias"]
	if annotationWithHost != "" {
		annotationWithHost = strings.TrimSpace(annotationWithHost)
		strings.Split(annotationWithHost, ",")
		for _, host := range strings.Split(annotationWithHost, ",") {
			host = strings.TrimSpace(host)
			if host != "" {
				hosts = append(hosts, strings.TrimSpace(host))
			}
		}
	}
	return hosts
}
func (p *IngressProvider) getConnectionTimeout(ctx context.Context, ingress *networkingv1.Ingress) time.Duration {
	logger := log.FromContext(ctx)
	defaultTimeout := time.Second * 5
	timeoutAnnotation := ingress.GetAnnotations()[annotationTimeout]
	if timeoutAnnotation == "" {
		return defaultTimeout
	}
	timeout, err := time.ParseDuration(timeoutAnnotation)
	if err != nil {
		logger.Warn(
			"failed to parse timeout annotation",
			log.Error(err),
			slog.String("namespace", ingress.GetNamespace()),
			slog.String("name", ingress.GetName()),
		)
		return defaultTimeout
	}
	return timeout
}
