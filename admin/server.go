package admin

import (
	"fmt"
	"net/http"
	"time"

	"github.com/arl/statsviz"
	"github.com/nakabonne/tstorage"
	"github.com/spf13/viper"
)

type AdminServer struct {
	TStorage *tstorage.Storage
}

func (srv *AdminServer) Init() {
	storage, _ := tstorage.NewStorage(
		tstorage.WithTimestampPrecision(tstorage.Seconds),
	)
	srv.TStorage = &storage
}

func (srv *AdminServer) Close() {
	(*srv.TStorage).Close()
}

func (srv *AdminServer) StartFakeApi() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", handler)

	statsviz.Register(mux)

	intf := fmt.Sprintf(":%d", viper.GetInt("admin.port"))
	fmt.Println("Listening on", intf)

	http.ListenAndServe(intf, mux)
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, time.Now().String())
}
