package operator

import (
	"context"
	"net/http"

	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/sturdy-journey/build"
	logging "github.com/ipfs/go-log/v2"
)

type Operator interface {
	Version(context.Context) (string, error)           //perm:read
	LogList(context.Context) ([]string, error)         //perm:write
	LogSetLevel(context.Context, string, string) error //perm:write
}

type OperatorImpl struct {
}

func (s *OperatorImpl) Version(ctx context.Context) (string, error) {
	return build.Version(), nil
}

func (s *OperatorImpl) LogList(ctx context.Context) ([]string, error) {
	return logging.GetSubsystems(), nil
}

func (s *OperatorImpl) LogSetLevel(ctx context.Context, subsystem string, level string) error {
	return logging.SetLogLevel(subsystem, level)
}

func NewOperatorClient(ctx context.Context, addr string, requestHeader http.Header) (Operator, jsonrpc.ClientCloser, error) {
	var res OperatorStruct
	closer, err := jsonrpc.NewMergeClient(ctx, addr, "Operator",
		[]interface{}{
			&res.Internal,
		},
		requestHeader,
	)

	return &res, closer, err
}

type OperatorStruct struct {
	Internal struct {
		Version     func(p0 context.Context) (string, error)             `perm:"read"`
		LogList     func(p0 context.Context) ([]string, error)           `perm:"write"`
		LogSetLevel func(p0 context.Context, p1 string, p2 string) error `perm:"write"`
	}
}

func (s *OperatorStruct) Version(p0 context.Context) (string, error) {
	return s.Internal.Version(p0)
}

func (s *OperatorStruct) LogList(p0 context.Context) ([]string, error) {
	return s.Internal.LogList(p0)
}

func (s *OperatorStruct) LogSetLevel(p0 context.Context, p1 string, p2 string) error {
	return s.Internal.LogSetLevel(p0, p1, p2)
}
