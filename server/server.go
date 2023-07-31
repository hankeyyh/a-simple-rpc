package server

type Server struct {
}

func NewServer() *Server {
	s := &Server{}
	return s
}

func (svr *Server) Register(service interface{}, metadata string) error {
	return nil
}

func (svr *Server) Serve(network, address string) error {
	return nil
}
