package health

import "context"

type Service struct {
	ctx context.Context
}

func NewService(ctx context.Context) *Service {
	return &Service{
		ctx: ctx,
	}
}

// Shutdown is now deprecated - context cancellation handles it
// Keeping for backward compatibility but it does nothing
func (s *Service) Shutdown() {
	// No-op - context cancellation handles shutdown
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
