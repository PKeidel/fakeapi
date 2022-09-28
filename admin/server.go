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
	"github.com/nakabonne/tstorage"
	"github.com/spf13/viper"
)

type AdminServer struct {
	vip      *viper.Viper
	tStorage *tstorage.Storage
}

func NewAdminServer(v *viper.Viper) *AdminServer {
	storage, _ := tstorage.NewStorage(
		tstorage.WithTimestampPrecision(tstorage.Milliseconds),
		tstorage.WithDataPath(v.GetString("admin.tstorage.data.path")),
	)
	return &AdminServer{vip: v, tStorage: &storage}
}

func (srv *AdminServer) Close() {
	(*srv.tStorage).Close()
	log.Println("AdminServer.Close() -> TStorage.Close()")
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

	intf := fmt.Sprintf(":%d", srv.vip.GetInt("admin.port"))
	fmt.Println("Listening on", intf)

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
		next.ServeHTTP(w, r)
		log.Println("Took:", time.Since(start))
		go func() {
			err := (*srv.tStorage).InsertRows([]tstorage.Row{
				{
					Metric:    "api.microseconds",
					DataPoint: tstorage.DataPoint{Timestamp: start.UnixMilli(), Value: float64(time.Since(start).Microseconds())},
					Labels: []tstorage.Label{
						{Name: "http.method", Value: r.Method},
						{Name: "http.path", Value: r.URL.Path},
					},
				},
			})
			log.Println("TStorage Insert Error:", err)
		}()
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
