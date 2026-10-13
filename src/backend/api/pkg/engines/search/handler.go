package search

import (
	"context"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgxpool"

	pb "quadsmith/api/gen/quadsmith"
)

// SearchServiceHandler implements quadsmithconnect.SearchServiceHandler.
type SearchServiceHandler struct {
	db *pgxpool.Pool
}

// NewSearchServiceHandler constructs a new SearchServiceHandler.
func NewSearchServiceHandler(db *pgxpool.Pool) *SearchServiceHandler {
	return &SearchServiceHandler{db: db}
}

// Search executes fuzzy text search across target collections with optional CEL filters.
func (s *SearchServiceHandler) Search(ctx context.Context, req *connect.Request[pb.SearchRequest]) (*connect.Response[pb.SearchResponse], error) {
	results, err := Search(ctx, s.db, req.Msg.GetQuery(), req.Msg.GetSelectors(), req.Msg.GetLimit())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&pb.SearchResponse{
		Results: results,
	}), nil
}
