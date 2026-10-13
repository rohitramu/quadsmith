package linkpreview

import (
	"context"

	"connectrpc.com/connect"

	pb "quadsmith/api/gen/quadsmith"
)

// LinkPreviewServiceHandler implements quadsmithconnect.LinkPreviewServiceHandler.
type LinkPreviewServiceHandler struct{}

// NewLinkPreviewServiceHandler constructs a new LinkPreviewServiceHandler.
func NewLinkPreviewServiceHandler() *LinkPreviewServiceHandler {
	return &LinkPreviewServiceHandler{}
}

// GetLinkPreview retrieves metadata and Open Graph information for a target URL.
func (h *LinkPreviewServiceHandler) GetLinkPreview(
	ctx context.Context,
	req *connect.Request[pb.GetLinkPreviewRequest],
) (*connect.Response[pb.GetLinkPreviewResponse], error) {
	targetURL := req.Msg.GetUrl()
	if targetURL == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, connect.NewError(connect.CodeInvalidArgument, nil))
	}

	preview, err := FetchLinkPreview(ctx, targetURL)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	return connect.NewResponse(preview), nil
}
