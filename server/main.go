package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/go-chassis/sc-client"
	"github.com/go-kratos/examples/helloworld/helloworld"
	prom "github.com/go-kratos/kratos/contrib/metrics/prometheus/v2"
	"github.com/go-kratos/kratos/contrib/registry/servicecomb/v2"
	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/metrics"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"os"
)

// go build -ldflags "-X main.Version=x.y.z"
var (
	// Name is the name of the compiled software.
	Name = "metrics"
	// Version is the version of the compiled software.
	// Version = "v1.0.0"

	_metricSeconds = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "server",
		Subsystem: "requests",
		Name:      "duration_sec",
		Help:      "server requests duration(sec).",
		Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.250, 0.5, 1},
	}, []string{"kind", "operation"})

	_metricRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "client",
		Subsystem: "requests",
		Name:      "code_total",
		Help:      "The total number of processed requests",
	}, []string{"kind", "operation", "code", "reason"})
)

func init() {
	prometheus.MustRegister(_metricSeconds, _metricRequests)
}

// server is used to implement helloworld.GreeterServer.
type server struct {
	helloworld.UnimplementedGreeterServer
}

// SayHello implements helloworld.GreeterServer
func (s *server) SayHello(ctx context.Context, in *helloworld.HelloRequest) (*helloworld.HelloReply, error) {
	return &helloworld.HelloReply{Message: fmt.Sprintf("Welcome %+v!", in.Name)}, nil
}

func main() {
	logPath := os.Getenv("LOG_PATH")
	if logPath == "" {
		logPath = "/logs/hello.log"
	}

	scEndPoint := os.Getenv("SC_ENDPOINT")
	if scEndPoint == "" {
		panic("sc endPoint is empty")
	}

	//logger := log.NewStdLogger(os.Stdout)
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0777)
	if err != nil {
		panic(err)
	}
	logger := log.NewStdLogger(logFile)
	log := log.NewHelper(logger)

	c, err := sc.NewClient(sc.Options{
		// EndPoints填写servicecomb中service center的地址
		Endpoints: []string{scEndPoint},
		EnableSSL: true,
		TLSConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	})
	if err != nil {
		panic(err)
	}
	r := servicecomb.NewRegistry(c)

	httpSrv := http.NewServer(
		http.Address(":8000"),
		http.Middleware(
			recovery.Recovery(),
			logging.Server(logger),
			metrics.Server(
				metrics.WithSeconds(prom.NewHistogram(_metricSeconds)),
				metrics.WithRequests(prom.NewCounter(_metricRequests)),
			),
		),
	)
	grpcSrv := grpc.NewServer(
		grpc.Address(":9000"),
		grpc.Middleware(
			recovery.Recovery(),
			logging.Server(logger),
			metrics.Server(
				metrics.WithSeconds(prom.NewHistogram(_metricSeconds)),
				metrics.WithRequests(prom.NewCounter(_metricRequests)),
			),
		),
	)
	httpSrv.Handle("/metrics", promhttp.Handler())

	s := &server{}
	helloworld.RegisterGreeterServer(grpcSrv, s)
	helloworld.RegisterGreeterHTTPServer(httpSrv, s)

	app := kratos.New(
		kratos.Name("helloworld"),
		kratos.Server(
			grpcSrv,
			httpSrv,
		),
		kratos.Registrar(r),
	)

	log.Info("start server helloword")

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
