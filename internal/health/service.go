package health

import "context"

type Service struct {
	ctx    context.Context
	cancel context.CancelFunc
}

func NewService() *Service {
	ctx, cancel := context.WithCancel(context.Background())
	return &Service{
		ctx:    ctx,
		cancel: cancel,
	}
}

func (s *Service) Shutdown() {
	s.cancel()
}

func (s *Service) IsShuttingDown() bool {
	select {
	case <-s.ctx.Done():
		return true
	default:
		return false
	}
}

// Context returns the service context for use in operations
func (s *Service) Context() context.Context {
	return s.ctx
}
