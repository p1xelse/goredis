package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"strconv"
	"sync"
	"time"
)

var ErrNotInteger = errors.New("value is not an integer")

type Storage struct {
	data   map[string]string
	dataMu sync.RWMutex

	expirationKeys   map[string]int64
	expirationKeysMu sync.RWMutex
}

func New(loadFromFileIfExist bool) (*Storage, error) {
	storage := &Storage{
		data:             map[string]string{},
		dataMu:           sync.RWMutex{},
		expirationKeys:   map[string]int64{},
		expirationKeysMu: sync.RWMutex{},
	}

	if loadFromFileIfExist {
		if err := storage.loadFromFileIfExist(); err != nil {
			return nil, fmt.Errorf("load from file if exist: %w", err)
		}
	}

	return storage, nil
}

func (s *Storage) Set(key string, value string, opts SetOptions) error {
	s.dataMu.Lock()
	s.data[key] = value
	s.dataMu.Unlock()

	var expirationTime int64
	if opts.TTL != nil {
		expirationTime = time.Now().Add(*opts.TTL).UnixMilli()
	}

	if opts.Expiration != nil {
		expirationTime = *opts.Expiration
	}

	if expirationTime > 0 {
		s.expirationKeysMu.Lock()
		s.expirationKeys[key] = expirationTime
		s.expirationKeysMu.Unlock()
	}

	return nil
}

func (s *Storage) Get(key string) (string, bool) {
	s.dataMu.RLock()
	defer s.dataMu.RUnlock()
	v, ok := s.data[key]
	return v, ok
}

func (s *Storage) Exists(keys []string) int {
	count := 0
	for i := 0; i < len(keys); i++ {
		key := keys[i]
		s.dataMu.RLock()
		if _, ok := s.data[key]; ok {
			count++
		}
		s.dataMu.RUnlock()
	}

	return count
}

func (s *Storage) Delete(keys []string) int {
	count := 0
	for i := 0; i < len(keys); i++ {
		key := keys[i]
		s.dataMu.Lock()
		if _, ok := s.data[key]; ok {
			count++
			delete(s.data, key)
		}
		s.dataMu.Unlock()
	}

	return count
}

func (s *Storage) Increment(key string) (int, error) {
	s.dataMu.Lock()
	defer s.dataMu.Unlock()
	numStr, ok := s.data[key]
	if !ok {
		numStr = "0"
	}
	num, err := strconv.ParseInt(numStr, 10, 64)
	if err != nil {
		return 0, ErrNotInteger
	}

	incrValue := num + 1

	s.data[key] = strconv.FormatInt(incrValue, 10)
	return int(incrValue), nil
}

func (s *Storage) Decrement(key string) (int, error) {
	s.dataMu.Lock()
	defer s.dataMu.Unlock()
	numStr, ok := s.data[key]
	if !ok {
		numStr = "0"
	}
	num, err := strconv.ParseInt(numStr, 10, 64)
	if err != nil {
		return 0, ErrNotInteger
	}

	decrValue := num - 1

	s.data[key] = strconv.FormatInt(decrValue, 10)
	return int(decrValue), nil
}

func (s *Storage) Save() error {
	infoForSave := saveInfo{}

	s.dataMu.Lock()
	infoForSave.Data = maps.Clone(s.data)
	s.dataMu.Unlock()

	s.expirationKeysMu.Lock()
	infoForSave.ExpirationKeys = maps.Clone(s.expirationKeys)
	s.expirationKeysMu.Unlock()

	file, err := os.Create("dump.json")
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(infoForSave); err != nil {
		return fmt.Errorf("failed to encode file: %v", err)
	}

	return nil
}

func (s *Storage) ExpireJob(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("expire job end: %v", ctx.Err())
		default:
		}

		for k, ex := range s.expirationKeys {
			if time.Now().UnixMilli() >= ex {
				s.dataMu.Lock()
				delete(s.expirationKeys, k)
				s.dataMu.Unlock()

				s.expirationKeysMu.Lock()
				delete(s.data, k)
				s.expirationKeysMu.Unlock()
			}
		}

		time.Sleep(100 * time.Millisecond)
	}
}

func (s *Storage) loadFromFileIfExist() error {
	file, err := os.Open("dump.json")
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("dump.json not found")
			// Файл не существует — это нормально
			return nil
		}
		return fmt.Errorf("could not open file: %v", err)
	}
	defer file.Close()

	var info saveInfo
	if err := json.NewDecoder(file).Decode(&info); err != nil {
		return fmt.Errorf("could not decode file: %v", err)
	}

	// Восстанавливаем данные
	s.dataMu.Lock()
	s.data = make(map[string]string)
	for k, v := range info.Data {
		s.data[k] = v
	}
	s.dataMu.Unlock()

	s.expirationKeysMu.Lock()
	s.expirationKeys = make(map[string]int64)
	for k, v := range info.ExpirationKeys {
		s.expirationKeys[k] = v
	}
	s.expirationKeysMu.Unlock()
	return nil
}
