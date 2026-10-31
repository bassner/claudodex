package proxy

import (
	"context"
	"fmt"
	"time"

	"github.com/bassner/claudodex/internal/auth"
)

func (s *Server) freshAuth(ctx context.Context) (auth.File, error) {
	file, err := s.cfg.AuthProvider.EnsureFresh(ctx, 5*time.Minute)
	if err != nil {
		return auth.File{}, err
	}
	return file, s.checkAuthIdentity(file)
}

func (s *Server) refreshAuth(ctx context.Context, rejected string) (auth.File, error) {
	file, err := s.cfg.AuthProvider.Refresh(ctx, rejected)
	if err != nil {
		return auth.File{}, err
	}
	return file, s.checkAuthIdentity(file)
}

func (s *Server) checkAuthIdentity(file auth.File) error {
	if (s.cfg.AuthAccountID != "" && file.Tokens.AccountID != s.cfg.AuthAccountID) ||
		(s.cfg.AuthUserID != "" && file.Tokens.UserID != s.cfg.AuthUserID) {
		return fmt.Errorf("%w: login changed; restart this session after signing in", auth.ErrInvalidAuth)
	}
	return nil
}
