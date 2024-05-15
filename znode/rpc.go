package znode

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"sync"
)

type Handler func(RpcServer, map[string]interface{}, context.Context) map[string]interface{}

type RpcServer struct {
	sync.RWMutex
	z            *Znode
	httpServer   *http.Server
	httpListener string
	timeout      int
	handlers     map[string]Handler
}

func NewRpcServer(z *Znode) *RpcServer {
	return &RpcServer{
		z:        z,
		handlers: make(map[string]Handler),
	}
}

func (z *Znode) startRpc() {
	rs := NewRpcServer(z)
	rs.Start()
}

func (rs *RpcServer) Start() error {
	rs.RegisterHandler("getWsAddr", getWsAddr)
	rs.RegisterHandler("getnodestates", getnodestates)
	rs.RegisterHandler("getNeighbors", getNeighbors)
	rs.RegisterHandler("getExtIp", getExtIp)

	port := strconv.Itoa(int(rs.z.config.RpcPort))
	http.HandleFunc("/rpc"+port, rs.rpcHandler)
	rs.httpListener = "127.0.0.1:" + port
	rs.httpServer = &http.Server{
		Addr: rs.httpListener,
	}
	go rs.httpServer.ListenAndServe()
	return nil
}

func (rs *RpcServer) rpcHandler(w http.ResponseWriter, r *http.Request) {
	rs.RLock()
	defer rs.RUnlock()

	w.Header().Add("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept")
	w.Header().Set("content-type", "application/json;charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != "POST" {
		w.Write([]byte("POST method is required"))
		return
	}

	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		w.Write([]byte(fmt.Sprintf(`{"error": "%v"}`, err)))
		return
	}

	req := make(map[string]interface{})
	err = json.Unmarshal(body, &req)
	if err != nil {
		w.Write([]byte(fmt.Sprintf(`{"error": "%v"}`, err)))
		return
	}

	if req["method"] == nil {
		w.Write([]byte(`{"error": "method is required"}`))
		return
	}

	req["remoteAddr"] = r.RemoteAddr[:len(r.RemoteAddr)-5]

	method := req["method"].(string)
	handler, ok := rs.handlers[method]
	if !ok {
		w.Write([]byte(fmt.Sprintf(`{"error": "method %s not found"}`, method)))
		return
	}
	data, err := json.Marshal(handler(*rs, req, r.Context()))
	if err != nil {
		w.Write([]byte(fmt.Sprintf(`{"error": "%v"}`, err)))
		return
	}
	w.Write(data)
}

func (rs *RpcServer) Stop() {
	rs.Lock()
	defer rs.Unlock()
	rs.httpServer.Close()
}

func (rs *RpcServer) RegisterHandler(method string, handler Handler) {
	rs.Lock()
	defer rs.Unlock()

	rs.handlers[method] = handler
}

func getWsAddr(rs RpcServer, params map[string]interface{}, ctx context.Context) map[string]interface{} {
	if len(params) == 0 {
		return map[string]interface{}{
			"error": "address is required",
		}
	}
	return nil
}

func getnodestates(rs RpcServer, params map[string]interface{}, ctx context.Context) map[string]interface{} {
	return nil
}

func getNeighbors(rs RpcServer, params map[string]interface{}, ctx context.Context) map[string]interface{} {
	res := make(map[string]interface{})
	for id, v := range rs.z.Neighbors {
		h := hex.EncodeToString([]byte(id))
		res[h] = v
	}
	return res
}

func getExtIp(rs RpcServer, params map[string]interface{}, ctx context.Context) map[string]interface{} {
	addr, ok := params["remoteAddr"].(string)
	if !ok {
		return map[string]interface{}{
			"error": "remoteAddr is required",
		}
	}
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return map[string]interface{}{
			"error": err.Error(),
		}
	}
	return map[string]interface{}{
		"extIp": host,
	}
}
