package client

import "context"

type XClient interface {
	Go(ctx context.Context, serviceMethod string, args interface{}, reply interface{}, done chan *Call) (*Call, error)
	Call(ctx context.Context, serviceMethod string, args interface{}, reply interface{}) error
	Close() error
}

type xclient struct {
}

func (x xclient) Go(ctx context.Context, serviceMethod string, args interface{}, reply interface{}, done chan *Call) (*Call, error) {
	//TODO implement me
	panic("implement me")
}

func (x xclient) Call(ctx context.Context, serviceMethod string, args interface{}, reply interface{}) error {
	//TODO implement me
	panic("implement me")
}

func (x xclient) Close() error {
	//TODO implement me
	panic("implement me")
}
