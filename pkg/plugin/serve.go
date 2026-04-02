package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/rpc"
	"os"
	"os/signal"
	"syscall"

	"github.com/hashicorp/go-hclog"
	hashicorpPlugin "github.com/hashicorp/go-plugin"
)

const PluginMagicCookieKey = "TOOLKIT_PLUGIN"
const PluginMagicCookieValue = "toolkit-v1"

// Serve starts the plugin as a hashicorp go-plugin managed process.
func Serve(tool Tool) {
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   tool.Metadata().Name,
		Level:  hclog.Debug,
		Output: os.Stderr,
	})

	hashicorpPlugin.Serve(&hashicorpPlugin.ServeConfig{
		HandshakeConfig: hashicorpPlugin.HandshakeConfig{
			ProtocolVersion:  1,
			MagicCookieKey:   PluginMagicCookieKey,
			MagicCookieValue: PluginMagicCookieValue,
		},
		Plugins: map[string]hashicorpPlugin.Plugin{
			"tool": &ToolkitRPCPlugin{Impl: tool},
		},
		Logger: logger,
	})
}

// ToolkitRPCPlugin implements hashicorp/go-plugin.Plugin for netrpc.
type ToolkitRPCPlugin struct {
	hashicorpPlugin.Plugin
	Impl Tool
}

func (p *ToolkitRPCPlugin) Server(*hashicorpPlugin.MuxBroker) (interface{}, error) {
	return &Plugin{Impl: p.Impl}, nil
}

func (p *ToolkitRPCPlugin) Client(b *hashicorpPlugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &RPCClient{client: c}, nil
}

// RPCArgs is the common argument wrapper for RPC calls.
type RPCArgs struct {
	Data map[string]string
}

// RPCReply is the common reply wrapper for RPC calls.
type RPCReply struct {
	Success bool
	Data    string
	Error   string
}

// Plugin implements the server-side RPC handlers.
// Named "Plugin" so hashicorp/go-plugin registers methods as "Plugin.MethodName".
type Plugin struct {
	Impl Tool
}

func (s *Plugin) GetMetadata(args struct{}, reply *ToolMetadata) error {
	*reply = s.Impl.Metadata()
	return nil
}

func (s *Plugin) Initialize(args *RPCArgs, reply *RPCReply) error {
	config := make(map[string]interface{})
	for k, v := range args.Data {
		var value interface{}
		if err := json.Unmarshal([]byte(v), &value); err != nil {
			config[k] = v
		} else {
			config[k] = value
		}
	}
	err := s.Impl.Init(context.Background(), config)
	if err != nil {
		reply.Success = false
		reply.Error = err.Error()
		return nil
	}
	reply.Success = true
	return nil
}

func (s *Plugin) Execute(args *RPCArgs, reply *RPCReply) error {
	params := make(map[string]interface{})
	for k, v := range args.Data {
		var value interface{}
		if err := json.Unmarshal([]byte(v), &value); err != nil {
			params[k] = v
		} else {
			params[k] = value
		}
	}
	result, err := s.Impl.Execute(context.Background(), params)
	if err != nil {
		reply.Success = false
		reply.Error = err.Error()
		return nil
	}
	dataBytes, err := json.Marshal(result.Data)
	if err != nil {
		reply.Success = false
		reply.Error = err.Error()
		return nil
	}
	reply.Success = result.Success
	reply.Data = string(dataBytes)
	reply.Error = result.Error
	return nil
}

func (s *Plugin) Validate(args *RPCArgs, reply *RPCReply) error {
	params := make(map[string]interface{})
	for k, v := range args.Data {
		var value interface{}
		if err := json.Unmarshal([]byte(v), &value); err != nil {
			params[k] = v
		} else {
			params[k] = value
		}
	}
	err := s.Impl.Validate(params)
	if err != nil {
		reply.Success = false
		reply.Error = err.Error()
		return nil
	}
	reply.Success = true
	return nil
}

func (s *Plugin) Shutdown(args struct{}, reply *RPCReply) error {
	err := s.Impl.Shutdown(context.Background())
	if err != nil {
		reply.Success = false
		reply.Error = err.Error()
		return nil
	}
	reply.Success = true
	return nil
}

// RPCClient implements the client-side RPC wrapper.
type RPCClient struct {
	client *rpc.Client
}

func (c *RPCClient) Metadata() ToolMetadata {
	var meta ToolMetadata
	err := c.client.Call("Plugin.GetMetadata", struct{}{}, &meta)
	if err != nil {
		return ToolMetadata{}
	}
	return meta
}

func (c *RPCClient) Init(ctx context.Context, config map[string]interface{}) error {
	data := make(map[string]string)
	for k, v := range config {
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			data[k] = fmt.Sprintf("%v", v)
		} else {
			data[k] = string(jsonBytes)
		}
	}
	var reply RPCReply
	err := c.client.Call("Plugin.Initialize", &RPCArgs{Data: data}, &reply)
	if err != nil {
		return err
	}
	if !reply.Success {
		return fmt.Errorf("%s", reply.Error)
	}
	return nil
}

func (c *RPCClient) Execute(ctx context.Context, params map[string]interface{}) (*Result, error) {
	data := make(map[string]string)
	for k, v := range params {
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			data[k] = fmt.Sprintf("%v", v)
		} else {
			data[k] = string(jsonBytes)
		}
	}
	var reply RPCReply
	err := c.client.Call("Plugin.Execute", &RPCArgs{Data: data}, &reply)
	if err != nil {
		return nil, err
	}

	var resultData interface{}
	if reply.Data != "" {
		if err := json.Unmarshal([]byte(reply.Data), &resultData); err != nil {
			resultData = reply.Data
		}
	}

	return &Result{
		Success: reply.Success,
		Data:    resultData,
		Error:   reply.Error,
	}, nil
}

func (c *RPCClient) Validate(params map[string]interface{}) error {
	data := make(map[string]string)
	for k, v := range params {
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			data[k] = fmt.Sprintf("%v", v)
		} else {
			data[k] = string(jsonBytes)
		}
	}
	var reply RPCReply
	err := c.client.Call("Plugin.Validate", &RPCArgs{Data: data}, &reply)
	if err != nil {
		return err
	}
	if !reply.Success {
		return fmt.Errorf("%s", reply.Error)
	}
	return nil
}

func (c *RPCClient) Shutdown(ctx context.Context) error {
	var reply RPCReply
	err := c.client.Call("Plugin.Shutdown", struct{}{}, &reply)
	if err != nil {
		return err
	}
	if !reply.Success {
		return fmt.Errorf("%s", reply.Error)
	}
	return nil
}

// StandaloneServe starts a standalone RPC plugin server on the given address.
func StandaloneServe(tool Tool, addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	server := rpc.NewServer()
	server.Register(&Plugin{Impl: tool})

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		lis.Close()
	}()

	for {
		conn, err := lis.Accept()
		if err != nil {
			select {
			case <-sigCh:
				return nil
			default:
				return err
			}
		}
		go server.ServeConn(conn)
	}
}
