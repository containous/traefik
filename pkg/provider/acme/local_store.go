package acme

import (
	"encoding/json"
	"io"
	"os"
	"sync"

	"github.com/traefik/traefik/v2/pkg/log"
	"github.com/traefik/traefik/v2/pkg/safe"
)

var _ Store = (*LocalStore)(nil)

// LocalStore Stores implementation for local file.
type LocalStore struct {
	saveDataChan chan map[string]*StoredData
	filename     string

	lock       sync.RWMutex
	storedData map[string]*StoredData
}

// NewLocalStore initializes a new LocalStore with a file name.
func NewLocalStore(filename string) *LocalStore {
	store := &LocalStore{filename: filename, saveDataChan: make(chan map[string]*StoredData)}
	store.listenSaveAction()
	return store
}

func (s *LocalStore) save(resolverName string, storedData *StoredData) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.storedData[resolverName] = storedData

	// we cannot pass s.storedData directly, map is reference type and as result
	// we can face with race condition, so we need to work with objects copy
	s.saveDataChan <- s.unSafeCopyOfStoredData()
}

func (s *LocalStore) get(resolverName string) (*StoredData, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.storedData == nil {
		s.storedData = map[string]*StoredData{}

		hasData, err := CheckFile(s.filename)
		if err != nil {
			return nil, err
		}

		if hasData {
			logger := log.WithoutContext().WithField(log.ProviderName, "acme")

			f, err := os.Open(s.filename)
			if err != nil {
				return nil, err
			}
			defer f.Close()

			file, err := io.ReadAll(f)
			if err != nil {
				return nil, err
			}

			if len(file) > 0 {
				if err := json.Unmarshal(file, &s.storedData); err != nil {
					return nil, err
				}
			}

			// Delete all certificates with no value
			var certificates []*CertAndStore
			for _, storedData := range s.storedData {
				for _, certificate := range storedData.Certificates {
					if len(certificate.Certificate.Certificate) == 0 || len(certificate.Key) == 0 {
						logger.Debugf("Deleting empty certificate %v for %v", certificate, certificate.Domain.ToStrArray())
						continue
					}
					certificates = append(certificates, certificate)
				}
				if len(certificates) < len(storedData.Certificates) {
					storedData.Certificates = certificates

					// we cannot pass s.storedData directly, map is reference type and as result
					// we can face with race condition, so we need to work with objects copy
					s.saveDataChan <- s.unSafeCopyOfStoredData()
				}
			}
		}
	}

	if s.storedData[resolverName] == nil {
		s.storedData[resolverName] = &StoredData{}
	}
	return s.storedData[resolverName], nil
}

// listenSaveAction listens to a chan to store ACME data in json format into `LocalStore.filename`.
func (s *LocalStore) listenSaveAction() {
	safe.Go(func() {
		logger := log.WithoutContext().WithField(log.ProviderName, "acme")
		for object := range s.saveDataChan {
			data, err := json.MarshalIndent(object, "", "  ")
			if err != nil {
				logger.Error(err)
			}

			err = os.WriteFile(s.filename, data, 0o600)
			if err != nil {
				logger.Error(err)
			}
		}
	})
}

// unSafeCopyOfStoredData creates maps copy of storedData. Is not thread safe, you should use `s.lock`.
func (s *LocalStore) unSafeCopyOfStoredData() map[string]*StoredData {
	result := map[string]*StoredData{}
	for k, v := range s.storedData {
		result[k] = v
	}
	return result
}

// GetAccount returns ACME Account.
func (s *LocalStore) GetAccount(resolverName string) (*Account, error) {
	storedData, err := s.get(resolverName)
	if err != nil {
		return nil, err
	}

	return storedData.Account, nil
}

// SaveAccount stores ACME Account.
func (s *LocalStore) SaveAccount(resolverName string, account *Account) error {
	storedData, err := s.get(resolverName)
	if err != nil {
		return err
	}

	storedData.Account = account
	s.save(resolverName, storedData)

	return nil
}

// GetCertificates returns ACME Certificates list.
func (s *LocalStore) GetCertificates(resolverName string) ([]*CertAndStore, error) {
	storedData, err := s.get(resolverName)
	if err != nil {
		return nil, err
	}

	return storedData.Certificates, nil
}

// SaveCertificates stores ACME Certificates list.
func (s *LocalStore) SaveCertificates(resolverName string, certificates []*CertAndStore) error {
	storedData, err := s.get(resolverName)
	if err != nil {
		return err
	}

	storedData.Certificates = certificates
	s.save(resolverName, storedData)

	return nil
}
