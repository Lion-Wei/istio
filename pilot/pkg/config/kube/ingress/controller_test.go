package ingress

import (
	"reflect"
	"testing"
	"time"

	// "k8s.io/client-go/kubernetes"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/fake"

	meshconfig "istio.io/api/mesh/v1alpha1"
	"istio.io/istio/pilot/pkg/model"
	"istio.io/istio/pilot/pkg/serviceregistry/kube"
)

const (
	resync           = 1 * time.Second
	defaultNamespace = "default"
	domainSuffix     = "foo.com"
)

func makeFakeKubeAPIController(mesh *meshconfig.MeshConfig) model.ConfigStoreCache {
	clientSet := fake.NewSimpleClientset()
	return NewController(clientSet, mesh, kube.ControllerOptions{
		WatchedNamespace: defaultNamespace,
		ResyncPeriod:     resync,
		DomainSuffix:     domainSuffix,
	})
}

func TestControllerGet(t *testing.T) {
	// TODO: full up this mesh
	mesh := &meshconfig.MeshConfig{
		// TODO: need test different ingress controller mode
		IngressControllerMode: meshconfig.MeshConfig_STRICT,
	}

	testCases := []struct {
		name         string
		ingressName  string
		getNamespace string
		getType      string
		getName      string
		expectResult *model.Config
	}{
		{"wrong ingress namespace", "ingress1", "default1",
			model.Gateway.Type, "ingress1-1-1", nil},
		{"wrong ingress name", "ingress1", defaultNamespace,
			model.Gateway.Type, "ingress2-1-1", nil},
		{"get ingress with GateWay type", "ingress1", defaultNamespace,
			model.Gateway.Type, "ingress1-1-1", &model.Config{
				ConfigMeta: model.ConfigMeta{
					Type:      model.Gateway.Type,
					Group:     model.Gateway.Group,
					Version:   model.Gateway.Version,
					Name:      "ingress1" + "-" + model.IstioIngressGatewayName,
					Namespace: model.IstioIngressNamespace,
					Domain:    domainSuffix,
				},
			}},
		{"get ingress with VirtualService type", "ingress1", defaultNamespace,
			model.VirtualService.Type, "ingress1-1-1", &model.Config{}},
	}
	for _, c := range testCases {
		t.Run(c.name, func(t *testing.T) {
			controller := makeFakeKubeAPIController(mesh)
			addIngress(t, controller, c.ingressName)
			res := controller.Get(c.getType, c.getName, c.getNamespace)
			if !reflect.DeepEqual(res, c.expectResult) {
				t.Errorf("expect: %v, \ngot: %v", c.expectResult, res)
			}
		})
	}
}

func addIngress(t *testing.T, cc model.ConfigStoreCache, ingressName string) {
	ingress := &extensionsv1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ingressName,
			Namespace: defaultNamespace,
		},
		Spec: extensionsv1beta1.IngressSpec{
			Backend: &extensionsv1beta1.IngressBackend{
				ServiceName: "foo",
				ServicePort: intstr.FromInt(80),
			},
			Rules: []extensionsv1beta1.IngressRule{
				{
					Host: "foo.bar.com",
					IngressRuleValue: extensionsv1beta1.IngressRuleValue{
						HTTP: &extensionsv1beta1.HTTPIngressRuleValue{
							Paths: []extensionsv1beta1.HTTPIngressPath{
								{
									Path: "/foo",
									Backend: extensionsv1beta1.IngressBackend{
										ServiceName: "foo",
										ServicePort: intstr.FromInt(80),
									},
								},
							},
						},
					},
				},
			},
		},
	}
	controller, ok := cc.(*controller)
	if !ok {
		t.Fatalf("wrong interface %v", cc)
	}
	if err := controller.informer.GetStore().Add(ingress); err != nil {
		t.Errorf("Add ingress to cache err: %v", err)
	}
}
