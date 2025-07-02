package health

import "sync"

type Service struct {
	shuttingDown      bool
	shuttingDownMutex sync.RWMutex
}

func NewService() *Service {
	return &Service{
		shuttingDownMutex: sync.RWMutex{},
		shuttingDown:      false,
	}
}

func (s *Service) Shutdown() {
	s.shuttingDownMutex.Lock()
	s.shuttingDown = true
	s.shuttingDownMutex.Unlock()
}

func (s *Service) IsShuttingDown() bool {
	s.shuttingDownMutex.RLock()
	defer s.shuttingDownMutex.RUnlock()
	return s.shuttingDown
}
