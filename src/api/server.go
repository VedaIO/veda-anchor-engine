package api

import (
	"database/sql"
	"sync"
	"veda-anchor-engine/src/internal/data/logger"
	"veda-anchor-engine/src/internal/data/repository"
	"veda-anchor-engine/src/internal/platform/nativehost"
)

// Server holds the dependencies for the API server, such as the database connection and the logger.
type Server struct {
	Logger          logger.Logger
	IsAuthenticated bool
	Mu              sync.Mutex
	db              *sql.DB
	Apps            *repository.AppRepository
	Web             *repository.WebRepository
}

// NewServer creates a new Server with its dependencies.
func NewServer(db *sql.DB) *Server {
	l := logger.GetLogger()
	return &Server{
		Logger: l,
		db:     db,
		Apps:   repository.NewAppRepository(db),
		Web:    repository.NewWebRepository(db),
	}
}

// GetWebDetails retrieves metadata for a given domain.
func (s *Server) GetWebDetails(domain string) (repository.WebMetadata, error) {
	meta, err := s.Web.GetMetadata(domain)
	if err != nil {
		return repository.WebMetadata{}, err
	}

	if meta == nil {
		return repository.WebMetadata{Domain: domain}, nil
	}

	return *meta, nil
}

// RegisterExtension handles the registration of the browser extension.
func (s *Server) RegisterExtension(id string) error {
	return nativehost.RegisterExtension(id)
}
