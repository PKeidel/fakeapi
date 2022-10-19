package server

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/PKeidel/fakeapi/frontend/dist"
	"github.com/PKeidel/fakeapi/router"
	"github.com/goji/httpauth"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	influxdb2api "github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/spf13/viper"
)

type FakeApiServer struct {
	vip            *viper.Viper
	InfluxClient   *influxdb2.Client
	InfluxWriteApi *influxdb2api.WriteAPIBlocking
	Routers        []router.FindRouter
}

func NewFakeApiServer(v *viper.Viper) *FakeApiServer {
	influxClient := influxdb2.NewClient(v.GetString("logging.metrics.influx.uri"), v.GetString("logging.metrics.influx.token"))
	writeAPI := influxClient.WriteAPIBlocking(v.GetString("logging.metrics.influx.org"), v.GetString("logging.metrics.influx.bucket"))
	routers := make([]router.FindRouter, 1)
	routers[0] = router.NewBasicRouter()
	return &FakeApiServer{
		vip:            v,
		InfluxClient:   &influxClient,
		InfluxWriteApi: &writeAPI,
		Routers:        routers,
	}
}

func (srv *FakeApiServer) Close() {
	log.Println("AdminServer.Close()")
	log.Println("  InfluxDbClient.Close()")
	(*srv.InfluxClient).Close()
}

func (srv *FakeApiServer) StartFakeApi() {
	mux := http.NewServeMux()

	authHandler := httpauth.SimpleBasicAuth(srv.vip.GetString("admin.username"), srv.vip.GetString("admin.password"))
	finalHandler := http.HandlerFunc(getHandler(srv))

	mux.Handle("/__admin/", http.StripPrefix("/__admin/", http.FileServer(http.FS(dist.AssetsFs))))

	mux.Handle(
		"/",
		authHandler(
			middlewareTimeLogging(
				srv,
				finalHandler,
			),
		),
	)

	// Open Stats in Browser on http://localhost:8080/debug/statsviz/
	// statsviz.Register(mux)

	intf := srv.vip.GetString("admin.listen")
	fmt.Printf("Listening on http://%s\n", intf)

	httpServer := http.Server{
		Addr:    intf,
		Handler: mux,
	}

	idleConnectionsClosed := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint
		if err := httpServer.Shutdown(context.Background()); err != nil {
			log.Printf("HTTP Server Shutdown Error: %v", err)
		}
		close(idleConnectionsClosed)
	}()

	if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("HTTP server ListenAndServe Error: %v", err)
	}

	<-idleConnectionsClosed

	log.Printf("Bye bye")
}

func middlewareTimeLogging(srv *FakeApiServer, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Log every incomming request
		log.Printf("%s %s %v", r.Method, r.URL.Path, r.URL.Query())

		next.ServeHTTP(w, r)

		if srv.vip.GetBool("logging.metrics.influx.enabled") {
			p := influxdb2.NewPointWithMeasurement("api.stats").
				AddTag("unit", "temperature").
				AddTag("http.method", r.Method).
				AddTag("http.path", r.URL.Path).
				AddField("microseconds", time.Since(start).Microseconds()).
				SetTime(time.Now())
			err := (*srv.InfluxWriteApi).WritePoint(context.Background(), p)
			if err != nil {
				log.Println("Influx Write Error:", err)
				srv.vip.Set("logging.metrics.influx.enabled", false)
				srv.vip.WriteConfig()
			}
		}
	})
}

func getHandler(srv *FakeApiServer) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {

		b, err := dist.AssetsFs.ReadFile(r.URL.Path)
		if err == nil {
			w.Write(b)
			return
		}

		// ANY /favicon.ico
		if r.URL.Path == "/favicon.ico" {
			w.WriteHeader(404)
			return
		}

		// POST /config
		if r.Method == http.MethodPost && r.URL.Path == "/config" {
			srv.vip.WriteConfig()
			w.WriteHeader(http.StatusCreated)
			return
		}

		// Query all registered routers
		for _, router := range srv.Routers {
			if respList, ok := router.FindRoutes(r); ok {
				randomIndex := rand.Intn(len(respList))
				resp := respList[randomIndex]
				if len(resp.ContentType) > 0 {
					w.Header().Add("Content-Type", resp.ContentType)
				}
				if resp.StatusCode > 0 {
					w.WriteHeader(resp.StatusCode)
				}
				if len(resp.Content) > 0 {
					fmt.Fprint(w, resp.Content)
				}
				return
			}
		}

		w.WriteHeader(404)
	}
}
