package main

import (
	"flag"
	"fmt"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/pluginpb"
	"strings"
)

const (
	contextPackage      = protogen.GoImportPath("context")
	rpcxServerPackage   = protogen.GoImportPath("github.com/hankeyyh/a-simple-rpc/server")
	rpcxClientPackage   = protogen.GoImportPath("github.com/hankeyyh/a-simple-rpc/client")
	rpcxProtocolPackage = protogen.GoImportPath("github.com/hankeyyh/a-simple-rpc/protocol")
)

func main() {

	protogen.Options{
		ParamFunc: flag.CommandLine.Set,
	}.Run(func(plugin *protogen.Plugin) error {
		plugin.SupportedFeatures = uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL)
		for _, f := range plugin.Files {
			if !f.Generate {
				continue
			}
			generateFile(plugin, f)
		}
		return nil
	})
}

func generateFile(gen *protogen.Plugin, file *protogen.File) *protogen.GeneratedFile {
	filename := file.GeneratedFilenamePrefix + "_simplerpc.pb.go"
	g := gen.NewGeneratedFile(filename, file.GoImportPath)
	g.P("// Code generated by protoc-gen-simple-rpc. DO NOT EDIT.")
	g.P()
	g.P("package ", file.GoPackageName)
	g.P()
	generateFileContent(gen, file, g)
	return g
}

func generateFileContent(gen *protogen.Plugin, file *protogen.File, g *protogen.GeneratedFile) {
	if len(file.Services) == 0 {
		return
	}

	g.P("// Reference imports to suppress errors if they are not otherwise used.")
	g.P("var _ = ", contextPackage.Ident("TODO"))
	// g.P("var _ = ", rpcxServerPackage.Ident("NewServer"))
	g.P("var _ = ", rpcxClientPackage.Ident("NewClient"))
	g.P("var _ = ", rpcxProtocolPackage.Ident("NewMessage"))
	g.P()

	for _, service := range file.Services {
		genService(gen, file, g, service)
	}
}

func genService(gen *protogen.Plugin, file *protogen.File, g *protogen.GeneratedFile, service *protogen.Service) {
	serviceName := upperFirstLatter(service.GoName)
	g.P("//================== interface skeleton ===================")
	g.P(fmt.Sprintf("type %s interface {", serviceName))
	g.P()
	for _, method := range service.Methods {
		generateMethod(g, method)
	}
	g.P("}")

	g.P()
	g.P("//================== client stub ===================")
	g.P(fmt.Sprintf(`// %[1]s is a client wrapped XClient.
		type %[1]sClient struct{
			xclient client.XClient
		}

		// new%[1]sClient wraps a XClient as %[1]sClient.
		// You can pass a shared XClient object created by NewXClientFor%[1]s.
		func new%[1]sClient(xclient client.XClient) *%[1]sClient {
			return &%[1]sClient{xclient: xclient}
		}

		// NewXClientFor%[1]s creates a XClient.
		// You can configure this client with more options such as etcd registry, serialize type, select algorithm and fail mode.
		func NewXClientFor%[1]s(addr string) (client.XClient, error) {
			d, err := client.NewPeer2PeerDiscovery("tcp@"+addr, "")
			if err != nil {
				return nil, err
			}
			
			opt := client.DefaultOption
			opt.SerializeType = protocol.ProtoBuffer

			xclient := client.NewXClient("%[1]s", client.FailTry, client.RoundRobin, d, opt)

			return xclient,nil
		}
	`, serviceName))
	for _, method := range service.Methods {
		generateClientCode(g, service, method)
	}
}

func generateClientCode(g *protogen.GeneratedFile, service *protogen.Service, method *protogen.Method) {
	methodName := upperFirstLatter(method.GoName)
	serviceName := upperFirstLatter(service.GoName)
	inType := g.QualifiedGoIdent(method.Input.GoIdent)
	outType := g.QualifiedGoIdent(method.Output.GoIdent)
	g.P(fmt.Sprintf(`func (c *%sClient) %s(ctx context.Context, args *%s)(reply *%s, err error){
			reply = &%s{}
			err = c.xclient.Call(ctx,"%s",args, reply)
			return reply, err
		}`, serviceName, methodName, inType, outType, outType, method.GoName))
}

func generateMethod(g *protogen.GeneratedFile, method *protogen.Method) {
	methodName := upperFirstLatter(method.GoName)
	inType := g.QualifiedGoIdent(method.Input.GoIdent)
	outType := g.QualifiedGoIdent(method.Output.GoIdent)
	g.P(fmt.Sprintf(`%[1]s(ctx context.Context, args *%[2]s, reply *%[3]s) (err error)`,
		methodName, inType, outType))
}

// 首字母转化成大写
func upperFirstLatter(s string) string {
	if len(s) == 0 {
		return ""
	}
	if len(s) == 1 {
		return strings.ToUpper(string(s[0]))
	}
	return strings.ToUpper(string(s[0])) + s[1:]
}
