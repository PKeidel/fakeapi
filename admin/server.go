package admin

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/arl/statsviz"
	"github.com/goji/httpauth"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	influxdb2api "github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/spf13/viper"
)

type AdminServer struct {
	vip            *viper.Viper
	InfluxClient   *influxdb2.Client
	InfluxWriteApi *influxdb2api.WriteAPIBlocking
}

func NewAdminServer(v *viper.Viper) *AdminServer {
	influxClient := influxdb2.NewClient(v.GetString("logging.metrics.influx.uri"), v.GetString("logging.metrics.influx.token"))
	writeAPI := influxClient.WriteAPIBlocking(v.GetString("logging.metrics.influx.org"), v.GetString("logging.metrics.influx.bucket"))
	return &AdminServer{
		vip:            v,
		InfluxClient:   &influxClient,
		InfluxWriteApi: &writeAPI,
	}
}

func (srv *AdminServer) Close() {
	log.Println("AdminServer.Close()")
	log.Println("  InfluxDbClient.Close()")
	(*srv.InfluxClient).Close()
}

func (srv *AdminServer) StartFakeApi() {
	mux := http.NewServeMux()

	authHandler := httpauth.SimpleBasicAuth(srv.vip.GetString("admin.username"), srv.vip.GetString("admin.password"))
	finalHandler := http.HandlerFunc(getHandler(srv))

	mux.Handle(
		"/",
		authHandler(
			middlewareTimeLogging(
				srv,
				finalHandler,
			),
		),
	)

	// Open Stats in Browser on http://localhost:9000/debug/statsviz/
	statsviz.Register(mux)

	intf := srv.vip.GetString("admin.port")
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

func middlewareTimeLogging(srv *AdminServer, next http.Handler) http.Handler {
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
			}
		}
	})
}

func getHandler(srv *AdminServer) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {

		if r.Method == http.MethodPost {
			srv.vip.WriteConfig()
		}

		fmt.Fprintf(w, "%s - %s (%s)", r.Method, r.URL.Path, time.Now().String())
	}
}
