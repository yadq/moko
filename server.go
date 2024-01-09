package main

import (
	"errors"
	"sync"
)

var ServerMap = serverMap{} // global server map

type Server interface {
	Init(cfgFile string) error
	Serve(*sync.WaitGroup) error
	Shutdown() error
}

type serverMap map[string]Server

func (m serverMap) Add(name string, server Server) {
	m[name] = server
}

func (m serverMap) Get(name string) (Server, error) {
	entry, exists := m[name]
	if !exists {
		return nil, errors.New("unknown protocol " + name)
	}

	return entry, nil
}

func (m serverMap) List() []string {
	names := make([]string, len(m))

	idx := 0
	for name := range m {
		names[idx] = name
		idx++
	}

	return names
}
