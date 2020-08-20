package rpc

import (
	"context"
	"github.com/azd1997/ecoin/enode"
	"net/http"
	"strconv"
)



const (
	// LocalHost "127.0.0.1"
	LocalHost = "127.0.0.1"
	// DefaultHTTPPort 23666
	DefaultHTTPPort = 23666

	version1Path  = "/v1"
	version2Path  = "/v2"
	GetRangeParam = "range"
	GetHashParam  = "hash"
	GetIDParam    = "id"
)

type Config struct {
	Port int
	En    *enode.Enode
}

// Server HTTP服务器，用于对本地的客户端提供查询等一些列API
// 监听IP： 127.0.0.1 （本地）
type Server struct {
	*http.Server
	en *enode.Enode
}

// 包内全局实例
var globalSvr *Server

type HTTPHandlers = []struct {
	Path string
	F    func(http.ResponseWriter, *http.Request)
}

func NewServer(conf *Config) *Server {
	sMux := http.NewServeMux()
	// tx
	for _, handler := range txHandlers {
		sMux.HandleFunc(handler.Path, handler.F)
	}
	// block
	for _, handler := range blockHandler {
		sMux.HandleFunc(handler.Path, handler.F)
	}
	// account
	for _, handler := range accountHandlers {
		sMux.HandleFunc(handler.Path, handler.F)
	}

	//default handler
	sMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	globalSvr = &Server{
		&http.Server{
			Addr:    LocalHost + ":" + strconv.Itoa(conf.Port),
			Handler: sMux,
		},
		conf.En,
	}

	return globalSvr
}

func (s *Server) Start() {
	go func() {
		if err := s.ListenAndServe(); err != http.ErrServerClosed {
			logger.Fatal("Http server listen failed:%v\n", err)
		}
	}()
}

func (s *Server) Stop() {
	if err := s.Shutdown(context.Background()); err != nil {
		logger.Warn("HTTP server shutdown err:%v\n", err)
	}
}
