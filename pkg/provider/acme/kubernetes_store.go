package acme

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"sync"

	"github.com/traefik/traefik/v2/pkg/log"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// FieldManager is the name of this process writing to k8s.
const FieldManager = "traefik"

// LabelResolver is the key of the Kubernetes label where we store the secret's
// resolver name.
const LabelResolver = "traefik.ingress.kubernetes.io/resolver"

// LabelACMEStorage is the key of the Kubernetes label that marks a sercet as
// stored.
const LabelACMEStorage = "traefik.ingress.kubernetes.io/acme-storage"

// KubernetesStore stores ACME account and certificates Kubernetes secrets.
// Each resolver gets it's own secrets and each domain is stored as a separate
// value in the secret.
// All secrets managed by this store well get the label
// `traefik.ingress.kubernetes.io/acme-storage=true`.
type KubernetesStore struct {
	ctx context.Context

	namespace string
	client    kubernetes.Interface

	mutex *sync.Mutex
	cache map[string]v1.Secret
}

// KubernetesStoreFromURI will create a new KubernetesStore instance from the
// given URI with this format: `kubernetes://:endpoint:/:namespace:`. The endpoint
// (or host:port part) of the uri is optional. Example: `kubernetes:///default`
func KubernetesStoreFromURI(uri string) (*KubernetesStore, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %q: %w", uri, err)
	}
	namespace := u.Path[1:]
	endpoint := ""
	if u.Host != "" {
		endpoint = u.Host
	}

	return NewKubernetesStore(namespace, endpoint)
}

// NewKubernetesStore will initiate a new KubernetesStore, create a Kubernetes
// clientset and start a resource watcher for stored sercrets.
// It will create a clientset with the default 'in-cluster' config.
func NewKubernetesStore(namespace, endpoint string) (*KubernetesStore, error) {
	store := &KubernetesStore{
		ctx:       context.Background(),
		namespace: namespace,
		mutex:     &sync.Mutex{},
		cache:     make(map[string]v1.Secret),
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create in-cluster configuration: %w", err)
	}
	if endpoint != "" {
		config.Host = endpoint
	}
	store.client, err = kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes clientset: %w", err)
	}

	go store.watcher()

	return store, nil
}

// GetAccount returns the account information for the given resolverName, this
// either from cache (which is maintained by the watcher and Save* operations)
// or it will fetch the resource fresh.
func (s *KubernetesStore) GetAccount(resolverName string) (*Account, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	secret, err := s.getSecretLocked(resolverName)
	if secret == nil || err != nil {
		return nil, err
	}

	accountData, found := secret.Data["account"]
	if !found || len(accountData) == 0 {
		return nil, nil
	}

	account := &Account{}
	err = json.Unmarshal(secret.Data["account"], account)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal account from secret data: %w", err)
	}

	return account, nil
}

// SaveAccount will patch the kubernetes secret resource for the given
// resolverName with the given account data. When the secret did not exist it is
// created with the correct labels set.
func (s *KubernetesStore) SaveAccount(resolverName string, account *Account) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	data, err := json.Marshal(account)
	if err != nil {
		return fmt.Errorf("failed to marshale account: %w", err)
	}

	patches := []patch{
		{
			Op:    "replace",
			Path:  "/data/account",
			Value: data,
		},
	}

	payload, _ := json.Marshal(patches)
	secret, err := s.client.CoreV1().Secrets(s.namespace).Patch(s.ctx, secretName(resolverName), types.JSONPatchType, payload, metav1.PatchOptions{FieldManager: FieldManager})

	status := &k8serrors.StatusError{}
	if err != nil && errors.As(err, &status) && status.Status().Code == 404 {
		secret = &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: secretName(resolverName),
				Labels: map[string]string{
					LabelACMEStorage: "true",
					LabelResolver:    resolverName,
				},
			},
			Data: map[string][]byte{
				"account": data,
			},
		}
		secret, err = s.client.CoreV1().Secrets(s.namespace).Create(s.ctx, secret, metav1.CreateOptions{FieldManager: FieldManager})
	}
	if err != nil {
		return fmt.Errorf("failed to patch secret: %w", err)
	}

	s.cache[resolverName] = *secret

	return nil
}

// GetCertificates returns all certificates for the given resolverName, this
// either from cache (which is maintained by the watcher and Save* operations)
// or it will fetch the resource fresh.
func (s *KubernetesStore) GetCertificates(resolverName string) ([]*CertAndStore, error) {
	logger := log.WithoutContext().WithField(log.ProviderName, "acme")

	s.mutex.Lock()
	defer s.mutex.Unlock()

	secret, err := s.getSecretLocked(resolverName)
	if secret == nil || err != nil {
		return nil, err
	}

	var result []*CertAndStore

	for domain, data := range secret.Data {
		if domain == "account" {
			continue
		}

		certAndStore := &CertAndStore{}
		err = json.Unmarshal(data, certAndStore)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal domain %q from secret data: %w", domain, err)
		}

		if domain != certAndStore.Domain.Main {
			logger.Warnf("mismatch in cert domain and secret keyname: %q != %q", domain, certAndStore.Domain.Main)
		}

		result = append(result, certAndStore)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Domain.Main < result[j].Domain.Main
	})

	return result, nil
}

// SaveCertificates will patch the kubernetes secret resource for the given
// resolverName with the given certificates. When the secret did not exist it is
// created with the correct labels set.
func (s *KubernetesStore) SaveCertificates(resolverName string, certs []*CertAndStore) error {
	logger := log.WithoutContext().WithField(log.ProviderName, "acme")

	s.mutex.Lock()
	defer s.mutex.Unlock()

	patches := []patch{}
	creationData := make(map[string][]byte)

	for _, cert := range certs {
		if cert.Domain.Main == "" {
			logger.Warn("not saving a certificate without a main domainname")

			continue
		}

		data, err := json.Marshal(cert)
		if err != nil {
			return fmt.Errorf("failed to marshale account: %w", err)
		}

		patches = append(patches, patch{
			Op:    "replace",
			Path:  "/data/" + cert.Domain.Main,
			Value: data,
		})

		creationData[cert.Domain.Main] = data
	}

	payload, _ := json.Marshal(patches)
	secret, err := s.client.CoreV1().Secrets(s.namespace).Patch(s.ctx, secretName(resolverName), types.JSONPatchType, payload, metav1.PatchOptions{})

	status := &k8serrors.StatusError{}
	if err != nil && errors.As(err, &status) && status.Status().Code == 404 {
		secret = &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: secretName(resolverName),
				Labels: map[string]string{
					LabelACMEStorage: "true",
					LabelResolver:    resolverName,
				},
			},
			Data: creationData,
		}
		secret, err = s.client.CoreV1().Secrets(s.namespace).Create(s.ctx, secret, metav1.CreateOptions{FieldManager: FieldManager})
	}

	if err != nil {
		return fmt.Errorf("failed to patch secret: %w", err)
	}

	s.cache[resolverName] = *secret

	return nil
}

func (s *KubernetesStore) watcher() {
	logger := log.WithoutContext().WithField(log.ProviderName, "acme")

	watcher, err := s.client.CoreV1().Secrets(s.namespace).Watch(s.ctx, metav1.ListOptions{
		Watch:         true,
		LabelSelector: LabelACMEStorage + "=true",
	})
	if err != nil {
		logger.Fatalf("failed to start a watch on kuberetes secrets for acme storage: %v", err)
	}
	defer watcher.Stop()

	for event := range watcher.ResultChan() {
		if event.Type != watch.Added && event.Type != watch.Modified {
			continue
		}

		secret, ok := event.Object.(*v1.Secret)
		if !ok {
			logger.Warn("kubernetes watch event was not of type secret")

			continue
		}
		resolver := secret.Labels[LabelResolver]
		if resolver != "" {
			s.mutex.Lock()
			s.cache[resolver] = *secret
			s.mutex.Unlock()
		}
	}
}

func (s *KubernetesStore) getSecretLocked(resolverName string) (*v1.Secret, error) {
	if _, found := s.cache[resolverName]; !found {
		secret, err := s.client.CoreV1().Secrets(s.namespace).Get(s.ctx, secretName(resolverName), metav1.GetOptions{})
		status := &k8serrors.StatusError{}
		if err != nil && errors.As(err, &status) && status.Status().Code == 404 {
			return nil, nil
		}
		if err != nil {
			return nil, fmt.Errorf("failed to fetch secret %q: %w", secretName(resolverName), err)
		}
		s.cache[resolverName] = *secret
	}
	secret := s.cache[resolverName]

	return &secret, nil
}

type patch struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value []byte `json:"value"`
}

func secretName(resolverName string) string {
	return "traefik-acme-" + resolverName + "-storage"
}
