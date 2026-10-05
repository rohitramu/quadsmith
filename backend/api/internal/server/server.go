package server

import (
	"context"
	"strings"
	"errors"

	"quadsmith/api/gen/quadsmith/quadsmithconnect"
	"quadsmith/api/internal/store"
	"quadsmith/engines/compatibility"
	"quadsmith/engines/compatibility/rules"
	"quadsmith/engines/evaluator"

	"connectrpc.com/connect"

	pb "quadsmith/api/gen/quadsmith"
)

type QuadsmithServer struct {
	compStore    *store.ComponentStore
	buildStore   *store.BuildStore
	evaluator    *evaluator.Evaluator
	compatEngine *compatibility.Engine
}

func NewQuadsmithServer(compStore *store.ComponentStore, buildStore *store.BuildStore) *QuadsmithServer {
	return &QuadsmithServer{
		compStore:  compStore,
		buildStore: buildStore,
		evaluator:  evaluator.NewEvaluator(),
		compatEngine: compatibility.NewEngine(
			&rules.VoltageRule{},
			&rules.MountRule{},
			&rules.UartRule{},
		),
	}
}

// Check interfaces
var _ quadsmithconnect.QuadsmithAPIHandler = (*QuadsmithServer)(nil)

// =======================
// COMPONENTS
// =======================

func (s *QuadsmithServer) ListComponents(
	ctx context.Context,
	req *connect.Request[pb.ListComponentsRequest],
) (*connect.Response[pb.ListComponentsResponse], error) {

	filter := req.Msg.GetFilter()
	if req.Msg.GetComponentType() != "" {
		typeFilter := "type == '" + strings.ToUpper(req.Msg.GetComponentType()) + "'"
		if filter == "" {
			filter = typeFilter
		} else {
			filter = "(" + filter + ") && " + typeFilter
		}
	}

	comps, err := s.compStore.SearchComponents(ctx, filter, req.Msg.GetFieldMask())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	res := &pb.ListComponentsResponse{}
	res.SetComponents(comps)
	res.SetTotalCount(int32(len(comps)))
	return connect.NewResponse(res), nil
}

func (s *QuadsmithServer) GetComponent(
	ctx context.Context,
	req *connect.Request[pb.GetComponentRequest],
) (*connect.Response[pb.Component], error) {
	filter := `id == "` + req.Msg.GetId() + `" || uuid == "` + req.Msg.GetId() + `" || id.endsWith("/` + req.Msg.GetId() + `")`
	comps, err := s.compStore.SearchComponents(ctx, filter, req.Msg.GetFieldMask())
	if err != nil || len(comps) == 0 {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("component not found"))
	}
	return connect.NewResponse(comps[0]), nil
}

func (s *QuadsmithServer) CreateComponent(
	ctx context.Context,
	req *connect.Request[pb.CreateComponentRequest],
) (*connect.Response[pb.Component], error) {
	if err := s.compStore.CreateComponent(ctx, req.Msg.GetComponent()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(req.Msg.GetComponent()), nil
}

func (s *QuadsmithServer) UpdateComponent(
	ctx context.Context,
	req *connect.Request[pb.UpdateComponentRequest],
) (*connect.Response[pb.Component], error) {
	if err := s.compStore.UpdateComponent(ctx, req.Msg.GetComponent()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(req.Msg.GetComponent()), nil
}

func (s *QuadsmithServer) DeleteComponent(
	ctx context.Context,
	req *connect.Request[pb.DeleteComponentRequest],
) (*connect.Response[pb.DeleteComponentResponse], error) {
	if err := s.compStore.DeleteComponent(ctx, req.Msg.GetId()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&pb.DeleteComponentResponse{}), nil
}

// =======================
// BUILDS
// =======================

func (s *QuadsmithServer) ListBuilds(
	ctx context.Context,
	req *connect.Request[pb.ListBuildsRequest],
) (*connect.Response[pb.ListBuildsResponse], error) {
	builds, err := s.buildStore.ListBuilds(ctx, req.Msg.GetFilter(), req.Msg.GetFieldMask())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse((&pb.ListBuildsResponse_builder{Builds: builds}).Build()), nil
}

func (s *QuadsmithServer) GetBuild(
	ctx context.Context,
	req *connect.Request[pb.GetBuildRequest],
) (*connect.Response[pb.Build], error) {
	build, err := s.buildStore.GetBuild(ctx, req.Msg.GetId(), req.Msg.GetFieldMask())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(build), nil
}

func (s *QuadsmithServer) CreateBuild(
	ctx context.Context,
	req *connect.Request[pb.CreateBuildRequest],
) (*connect.Response[pb.Build], error) {
	if err := s.buildStore.CreateBuild(ctx, req.Msg.GetBuild_()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(req.Msg.GetBuild_()), nil
}

func (s *QuadsmithServer) UpdateBuild(
	ctx context.Context,
	req *connect.Request[pb.UpdateBuildRequest],
) (*connect.Response[pb.Build], error) {
	if err := s.buildStore.UpdateBuild(ctx, req.Msg.GetBuild_()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(req.Msg.GetBuild_()), nil
}

func (s *QuadsmithServer) DeleteBuild(
	ctx context.Context,
	req *connect.Request[pb.DeleteBuildRequest],
) (*connect.Response[pb.DeleteBuildResponse], error) {
	if err := s.buildStore.DeleteBuild(ctx, req.Msg.GetId()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&pb.DeleteBuildResponse{}), nil
}

// =======================
// ENGINES
// =======================

func (s *QuadsmithServer) EvaluateBuild(
	ctx context.Context,
	req *connect.Request[pb.EvaluateBuildRequest],
) (*connect.Response[pb.EvaluateBuildResponse], error) {
	var buildComponents []*pb.Component
	var battery *pb.Component

	for _, id := range req.Msg.GetComponentIds() {
		comps, err := s.compStore.SearchComponents(ctx, `id == "`+id+`" || uuid == "`+id+`" || id.endsWith("/`+id+`")`, nil)
		if err == nil && len(comps) > 0 {
			if comps[0].WhichType() == pb.Component_Battery_case {
				battery = comps[0]
			} else {
				buildComponents = append(buildComponents, comps[0])
			}
		}
	}

	resp, err := s.evaluator.Evaluate(buildComponents, battery, req.Msg.GetPayloadWeightG())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(resp), nil
}

func (s *QuadsmithServer) CheckCompatibility(
	ctx context.Context,
	req *connect.Request[pb.CheckCompatibilityRequest],
) (*connect.Response[pb.CheckCompatibilityResponse], error) {

	var resolved []*pb.Component
	for _, id := range req.Msg.GetComponentIds() {
		comps, err := s.compStore.SearchComponents(ctx, `id == "`+id+`" || uuid == "`+id+`" || id.endsWith("/`+id+`")`, nil)
		if err == nil && len(comps) > 0 {
			resolved = append(resolved, comps[0])
		}
	}

	resp, err := s.compatEngine.Evaluate(req.Msg, resolved)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(resp), nil
}
