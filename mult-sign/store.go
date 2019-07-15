package main

import (
	"context"
	"io/ioutil"
	"os"
	"time"

	jsoniter "github.com/json-iterator/go"
	log "github.com/sirupsen/logrus"
)

// Store store
type Store struct {
	Path string

	batch uint64
}

func newStore(path string) (*Store, error) {
	s := &Store{
		Path: path,
	}

	if err := s.readBatch(); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Store) readFile(f string, v interface{}) (interface{}, error) {
	b, err := ioutil.ReadFile(s.Path + "/" + f)
	if err != nil {
		if os.IsNotExist(err) {
			err = nil
		}
		return nil, err
	}
	if err := jsoniter.Unmarshal(b, &v); err != nil {
		return nil, err
	}
	return v, nil
}

func (s *Store) writeFile(f string, v interface{}) error {
	b, err := jsoniter.Marshal(v)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(s.Path+f, b, 0644)
}

// Property property
type Property struct {
	UpdateAt time.Time `json:"updated_at"`
	Key      string    `json:"key"`
	Value    string    `json:"property"`
}

// ReadProperty read property
func (s *Store) ReadProperty(ctx context.Context, key string) (string, error) {
	log.Debugln("ReadProperty", key)
	p, err := s.readFile(key+".json", &Property{})
	if p == nil || err != nil {
		return "", err
	}
	return p.(*Property).Value, nil
}

// WriteProperty write property
func (s *Store) WriteProperty(ctx context.Context, key, value string) error {
	log.Debugln("WriteProperty", key, value)
	p := Property{
		UpdateAt: time.Now(),
		Key:      key,
		Value:    value,
	}
	return s.writeFile(key+".json", p)
}

// Batch batch
type Batch struct {
	UpdateAt time.Time `json:"updated_at"`
	Batch    uint64    `json:"batch"`
}

func (s *Store) writeBatch(batch uint64) error {
	s.batch = batch

	b := Batch{
		UpdateAt: time.Now(),
		Batch:    s.batch,
	}
	return s.writeFile("batch.json", b)
}

func (s *Store) readBatch() error {
	d, err := s.readFile("batch.json", &Batch{})

	if d == nil || err != nil {
		return err
	}
	s.batch = d.(*Batch).Batch
	return nil
}

// Batch batch
func (s *Store) Batch() uint64 {
	return s.batch
}
