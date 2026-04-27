package server

import (
	"bufio"
	"context"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/whoiscache/whoiscache/internal/config"
	"github.com/whoiscache/whoiscache/internal/metrics"
	"github.com/whoiscache/whoiscache/internal/service"
)

type TCPServer struct {
	cfg     *config.Config
	svc     *service.QueryService
	sem     chan struct{}
	ln      net.Listener
	metrics *http.Server
}

func NewTCPServer(cfg *config.Config, svc *service.QueryService) *TCPServer {
	w := cfg.Server.WorkerPoolSize
	if w <= 0 {
		w = 256
	}
	return &TCPServer{
		cfg: cfg,
		svc: svc,
		sem: make(chan struct{}, w),
	}
}

func (s *TCPServer) Listen() error {
	ln, err := net.Listen("tcp", s.cfg.Server.ListenAddr)
	if err != nil {
		return err
	}
	s.ln = ln
	return nil
}

func (s *TCPServer) Addr() string {
	if s.ln == nil {
		return ""
	}
	return s.ln.Addr().String()
}

func (s *TCPServer) Serve(ctx context.Context) error {
	if s.ln == nil {
		if err := s.Listen(); err != nil {
			return err
		}
	}

	ma := s.cfg.Metrics.ListenAddr
	if ma == "" {
		ma = ":8080"
	}
	mux := http.NewServeMux()
	mux.Handle("/metrics", metrics.Handler())
	s.metrics = &http.Server{Addr: ma, Handler: mux}
	go func() { _ = s.metrics.ListenAndServe() }()

	go func() {
		<-ctx.Done()
		_ = s.ln.Close()
	}()

	var n int32
	for {
		c, err := s.ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return s.shutdown()
			}
			continue
		}
		if atomic.LoadInt32(&n) >= int32(s.cfg.Server.MaxConns) {
			_ = c.Close()
			metrics.Errors.WithLabelValues("max_conns").Inc()
			continue
		}
		atomic.AddInt32(&n, 1)
		s.sem <- struct{}{}
		go func(conn net.Conn) {
			defer func() {
				<-s.sem
				atomic.AddInt32(&n, -1)
			}()
			s.handle(ctx, conn)
		}(c)
	}
}

func (s *TCPServer) handle(_ context.Context, c net.Conn) {
	defer func() { _ = c.Close() }()
	_ = c.SetReadDeadline(time.Now().Add(s.cfg.Server.ReadTimeout))
	_ = c.SetWriteDeadline(time.Now().Add(s.cfg.Server.WriteTimeout))

	br := bufio.NewReader(c)
	line, err := br.ReadString('\n')
	if err != nil {
		metrics.Errors.WithLabelValues("read").Inc()
		return
	}
	total := s.cfg.Whois.DefaultTimeout * time.Duration(1+s.cfg.Whois.MaxReferralHops)
	reqCtx, cancel := context.WithTimeout(context.Background(), total)
	defer cancel()
	t0 := time.Now()
	kind := s.svc.QueryKind(line)
	b, lerr := s.svc.Lookup(reqCtx, line)
	if lerr != nil {
		metrics.Errors.WithLabelValues("lookup").Inc()
		metrics.Requests.WithLabelValues(lookupResultKind(kind), "error").Inc()
		_, _ = c.Write(lookupErrorText(line, lerr))
		return
	}
	metrics.Requests.WithLabelValues(lookupResultKind(kind), "ok").Inc()
	metrics.Duration.WithLabelValues(lookupResultKind(kind)).Observe(time.Since(t0).Seconds())
	if len(b) > 0 && b[len(b)-1] == '\n' {
		_, _ = c.Write(b)
		return
	}
	_, _ = c.Write(b)
	_, _ = c.Write([]byte("\r\n"))
}

func lookupResultKind(k string) string {
	if k == "invalid" {
		return "invalid"
	}
	if k == "" {
		return "unknown"
	}
	return k
}

func lookupErrorText(query string, _ error) []byte {
	_ = query
	return []byte("%Error: whois request failed (try again later)\r\n")
}

func (s *TCPServer) shutdown() error {
	if s.metrics != nil {
		_ = s.metrics.Shutdown(context.Background())
	}
	if s.ln != nil {
		_ = s.ln.Close()
	}
	return nil
}

func (s *TCPServer) Close() error {
	if s.ln == nil {
		return nil
	}
	return s.ln.Close()
}
