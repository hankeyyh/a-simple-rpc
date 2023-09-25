package server_plugin

// consul服务发现对应服务端的plugin
type ConsulPlugin struct {
}

func (c *ConsulPlugin) Register(name string, rcvr interface{}, metadata string) error {
	//TODO implement me
	panic("implement me")
}

func (c *ConsulPlugin) UnRegister(name string) error {
	//TODO implement me
	panic("implement me")
}

func (c *ConsulPlugin) Start() error {
	return nil
}

func (c *ConsulPlugin) Stop() error {
	return nil
}
