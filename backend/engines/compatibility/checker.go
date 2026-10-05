package compatibility

import (
	pb "quadsmith/api/gen/quadsmith"
)

// Rule is the interface that all compatibility checkers must implement.
type Rule interface {
	Name() string
	Check(components []*pb.Component) *pb.CompatibilityResult
}

type Engine struct {
	rules []Rule
}

// NewEngine initializes the compatibility engine with all available rules.
func NewEngine(rules ...Rule) *Engine {
	return &Engine{rules: rules}
}

// Evaluate runs all registered compatibility rules against a set of components.
func (e *Engine) Evaluate(req *pb.CheckCompatibilityRequest, resolvedComponents []*pb.Component) (*pb.CheckCompatibilityResponse, error) {
	var results []*pb.CompatibilityResult

	for _, rule := range e.rules {
		if res := rule.Check(resolvedComponents); res != nil {
			// Only append if the rule actually produced messages or metrics
			if len(res.GetMessages()) > 0 || res.GetMetrics() != nil {
				results = append(results, res)
			}
		}
	}

	return (&pb.CheckCompatibilityResponse_builder{
		Results: results,
	}).Build(), nil
}
