package acme

import (
	"context"
	"encoding/json"
	"math/rand"
	"os"
	"reflect"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/traefik/traefik/v2/pkg/types"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/clientcmd"
)

func setup(t *testing.T) (*KubernetesStore, string) {
	t.Helper()

	namespace := "traefik-test-" + strconv.FormatInt(rand.Int63(), 36)[2:]

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	store := &KubernetesStore{
		ctx:       ctx,
		namespace: namespace,
		mutex:     &sync.Mutex{},
		cache:     make(map[string]v1.Secret),
	}

	if os.Getenv("USE_MINIKUBE") != "" {
		config, err := clientcmd.BuildConfigFromFlags("", os.ExpandEnv("$HOME/.kube/config"))
		if err != nil {
			t.Fatalf("failed to open kubernetes config: %v", err)
		}
		store.client, err = kubernetes.NewForConfig(config)
		if err != nil {
			t.Fatalf("failed to create kubernetes client: %v", err)
		}
	} else {
		client := fake.NewSimpleClientset()
		store.client = client
	}

	_, _ = store.client.CoreV1().Namespaces().Create(context.Background(), &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}, metav1.CreateOptions{})

	t.Cleanup(func() {
		_ = store.client.CoreV1().Namespaces().Delete(context.Background(), namespace, metav1.DeleteOptions{})
	})

	return store, namespace
}

func TestKubernetesStore_SaveAccount(t *testing.T) {
	store, namespace := setup(t)

	account := &Account{
		Email:      "john@example.org",
		PrivateKey: []byte("0123456789"),
		KeyType:    certcrypto.RSA2048,
	}

	t.Run("create", func(t *testing.T) {
		err := store.SaveAccount("resolver01", account)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		secret, err := store.client.CoreV1().Secrets(namespace).Get(store.ctx, "traefik-acme-resolver01-storage", metav1.GetOptions{})
		if err != nil {
			t.Fatalf("failed to get secret: %v", err)
		}

		stored := &Account{}
		_ = json.Unmarshal(secret.Data["account"], stored)
		if !reflect.DeepEqual(account, stored) {
			t.Errorf("expected account %v, got %v instead", account, stored)
		}
	})

	t.Run("edit", func(t *testing.T) {
		account.Email = "jane@example.org"

		err := store.SaveAccount("resolver01", account)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		secret, err := store.client.CoreV1().Secrets(namespace).Get(store.ctx, "traefik-acme-resolver01-storage", metav1.GetOptions{})
		if err != nil {
			t.Fatalf("failed to get secret: %v", err)
		}

		stored := &Account{}
		_ = json.Unmarshal(secret.Data["account"], stored)
		if !reflect.DeepEqual(account, stored) {
			t.Errorf("expected account %v, got %v instead", account, stored)
		}
	})
}

func TestKubernetesStore_GetAccount(t *testing.T) {
	store, _ := setup(t)

	account := &Account{
		Email:      "john@example.org",
		PrivateKey: []byte("0123456789"),
		KeyType:    certcrypto.RSA2048,
	}

	err := store.SaveAccount("resolver01", account)
	if err != nil {
		t.Fatalf("failed to save account: %v", err)
	}
	store.cache = make(map[string]v1.Secret) // reset cache after saving

	t.Run("without cache", func(t *testing.T) {
		got, err := store.GetAccount("resolver01")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !reflect.DeepEqual(account, got) {
			t.Errorf("expected account %v, got %v instead", account, got)
		}
	})

	t.Run("with cache", func(t *testing.T) {
		got, err := store.GetAccount("resolver01")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !reflect.DeepEqual(account, got) {
			t.Errorf("expected account %v, got %v instead", account, got)
		}
	})
}

func TestKubernetesStore_GetAccount_NoSecret(t *testing.T) {
	store, _ := setup(t)

	account, err := store.GetAccount("resolver01")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if account != nil {
		t.Errorf("unexpected account: %v", account)
	}
}

func TestKubernetesStore_SaveCertificates(t *testing.T) {
	store, namespace := setup(t)

	certs := []*CertAndStore{
		{
			Certificate: Certificate{
				Domain:      types.Domain{Main: "example.org"},
				Certificate: []byte("0123456789"),
				Key:         []byte("0123456789"),
			},
			Store: "store01",
		},
		{
			Certificate: Certificate{
				Domain:      types.Domain{Main: "sub.example.org"},
				Certificate: []byte("9876543210"),
				Key:         []byte("9876543210"),
			},
			Store: "store02",
		},
	}

	t.Run("create", func(t *testing.T) {
		err := store.SaveCertificates("resolver01", certs)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		secret, err := store.client.CoreV1().Secrets(namespace).Get(store.ctx, "traefik-acme-resolver01-storage", metav1.GetOptions{})
		if err != nil {
			t.Fatalf("failed to get secret: %v", err)
		}

		stored := &CertAndStore{}
		_ = json.Unmarshal(secret.Data["example.org"], stored)
		if !reflect.DeepEqual(certs[0], stored) {
			t.Errorf("expected example.org certificated to be %v, got %v instead", certs[0], stored)
		}

		stored = &CertAndStore{}
		_ = json.Unmarshal(secret.Data["sub.example.org"], stored)
		if !reflect.DeepEqual(certs[1], stored) {
			t.Errorf("expected sub.example.org certificated to be %v, got %v instead", certs[1], stored)
		}
	})

	account := &Account{
		Email: "jane@example.org",
	}
	err := store.SaveAccount("resolver01", account)
	if err != nil {
		t.Fatalf("failed to store account: %v", err)
	}

	certs = append(certs, &CertAndStore{
		Certificate: Certificate{
			Domain:      types.Domain{Main: "docs.traefik.io"},
			Certificate: []byte("0123456789"),
			Key:         []byte("0123456789"),
		},
		Store: "store03",
	})

	t.Run("edit", func(t *testing.T) {
		err := store.SaveCertificates("resolver01", certs)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		secret, err := store.client.CoreV1().Secrets(namespace).Get(store.ctx, "traefik-acme-resolver01-storage", metav1.GetOptions{})
		if err != nil {
			t.Fatalf("failed to get secret: %v", err)
		}

		stored := &CertAndStore{}
		_ = json.Unmarshal(secret.Data["example.org"], stored)
		if !reflect.DeepEqual(certs[0], stored) {
			t.Errorf("expected example.org certificated to be %v, got %v instead", certs[0], stored)
		}

		stored = &CertAndStore{}
		_ = json.Unmarshal(secret.Data["sub.example.org"], stored)
		if !reflect.DeepEqual(certs[1], stored) {
			t.Errorf("expected sub.example.org certificated to be %v, got %v instead", certs[1], stored)
		}

		stored = &CertAndStore{}
		_ = json.Unmarshal(secret.Data["docs.traefik.io"], stored)
		if !reflect.DeepEqual(certs[2], stored) {
			t.Errorf("expected docs.traefik.io certificated to be %v, got %v instead", certs[2], stored)
		}

		got, err := store.GetAccount("resolver01")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !reflect.DeepEqual(account, got) {
			t.Errorf("expected account %v, got %v instead", account, got)
		}
	})
}

func TestKubernetesStore_GetCertificates(t *testing.T) {
	store, _ := setup(t)

	certs := []*CertAndStore{
		{
			Certificate: Certificate{
				Domain:      types.Domain{Main: "example.org"},
				Certificate: []byte("0123456789"),
				Key:         []byte("0123456789"),
			},
			Store: "store01",
		},
		{
			Certificate: Certificate{
				Domain:      types.Domain{Main: "sub.example.org"},
				Certificate: []byte("9876543210"),
				Key:         []byte("9876543210"),
			},
			Store: "store02",
		},
	}

	err := store.SaveCertificates("resolver01", certs)
	if err != nil {
		t.Fatalf("failed to save certificates: %v", err)
	}

	store.cache = make(map[string]v1.Secret) // reset cache after saving

	t.Run("without cache", func(t *testing.T) {
		got, err := store.GetCertificates("resolver01")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !reflect.DeepEqual(certs, got) {
			t.Errorf("expected certs %v, got %v instead", certs, got)
		}
	})

	t.Run("with cache", func(t *testing.T) {
		got, err := store.GetCertificates("resolver01")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !reflect.DeepEqual(certs, got) {
			t.Errorf("expected certs %v, got %v instead", certs, got)
		}
	})
}

func TestKubernetesStore_watcher(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	store, namespace := setup(t)

	if _, ok := store.client.(*fake.Clientset); ok {
		// fake.Clientset doesn't trigger watch events.
		t.Skip()
	}

	go store.watcher()

	if len(store.cache) != 0 {
		panic("wut?")
	}

	// manually creating the secret so we don't fill the cache:
	manualSecret, err := store.client.CoreV1().Secrets(namespace).Create(store.ctx, &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: secretName("resolver01"),
			Labels: map[string]string{
				LabelACMEStorage: "true",
				LabelResolver:    "resolver01",
			},
		},
		Data: map[string][]byte{
			"account": []byte(`{"Email":"john@example.org","Registration":null,"PrivateKey":"MDEyMzQ1Njc4OQ==","KeyType":"2048"}`),
		},
	}, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("failed to create secret: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	cachedSecret, found := store.cache["resolver01"]
	if !found {
		t.Fatal("no cached created by watcher")
	}

	if !reflect.DeepEqual(manualSecret.Data, cachedSecret.Data) {
		t.Error("stored and cached data is not the same")
	}
}
