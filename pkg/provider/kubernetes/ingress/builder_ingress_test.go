package ingress

import networkingv1 "k8s.io/api/networking/v1"

func buildIngress(opts ...func(*networkingv1.Ingress)) *networkingv1.Ingress {
	i := &networkingv1.Ingress{}
	i.Kind = "Ingress"
	for _, opt := range opts {
		opt(i)
	}
	return i
}

func iNamespace(value string) func(*networkingv1.Ingress) {
	return func(i *networkingv1.Ingress) {
		i.Namespace = value
	}
}

func iRules(opts ...func(*networkingv1.IngressSpec)) func(*networkingv1.Ingress) {
	return func(i *networkingv1.Ingress) {
		s := &networkingv1.IngressSpec{}
		for _, opt := range opts {
			opt(s)
		}
		i.Spec = *s
	}
}

func iRule(opts ...func(*networkingv1.IngressRule)) func(*networkingv1.IngressSpec) {
	return func(spec *networkingv1.IngressSpec) {
		r := &networkingv1.IngressRule{}
		for _, opt := range opts {
			opt(r)
		}
		spec.Rules = append(spec.Rules, *r)
	}
}

func iHost(name string) func(*networkingv1.IngressRule) {
	return func(rule *networkingv1.IngressRule) {
		rule.Host = name
	}
}

func iTLSes(opts ...func(*networkingv1.IngressTLS)) func(*networkingv1.Ingress) {
	return func(i *networkingv1.Ingress) {
		for _, opt := range opts {
			iTLS := networkingv1.IngressTLS{}
			opt(&iTLS)
			i.Spec.TLS = append(i.Spec.TLS, iTLS)
		}
	}
}

func iTLS(secret string, hosts ...string) func(*networkingv1.IngressTLS) {
	return func(i *networkingv1.IngressTLS) {
		i.SecretName = secret
		i.Hosts = hosts
	}
}
