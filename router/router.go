package router

import (
	_ "embed"
	"net/http"
)

func NewBasicRouter() BasicRouter {
	router := BasicRouter{}
	router.RoutesCache = make(RoutesCache)
	router.AddRoute("/api/users", "GET", RouterResponse{StatusCode: 200, ContentType: "application/json", Content: "[{\"id\":1,\"name\":\"admin\"}]"})
	router.AddRoute("/api/users", "GET", RouterResponse{StatusCode: 200, ContentType: "application/json", Content: "[{\"id\":2,\"name\":\"PKeidel\"}]"})
	return router
}

type FindRouter interface {
	FindRoutes(req *http.Request) ([]RouterResponse, bool)
}

type BasicRouter struct {
	RoutesCache RoutesCache
}

type RouterResponse struct {
	StatusCode  int
	Content     string
	ContentType string
}

func (router BasicRouter) AddRoute(path, method string, res RouterResponse) {
	if _, ok := router.RoutesCache[path]; !ok {
		router.RoutesCache[path] = make(map[string][]RouterResponse)
	}
	if _, ok := router.RoutesCache[path][method]; !ok {
		router.RoutesCache[path][method] = make([]RouterResponse, 0)
	}
	router.RoutesCache[path][method] = append(router.RoutesCache[path][method], res)
}

func (router BasicRouter) FindRoutes(req *http.Request) (res []RouterResponse, ok bool) {

	if _, ok := router.RoutesCache[req.URL.Path]; ok {
		if _, ok := router.RoutesCache[req.URL.Path][req.Method]; ok {
			if responses, ok := router.RoutesCache[req.URL.Path][req.Method]; ok {
				if len(responses) > 0 {
					res = append(res, responses...)
				}
			}
		}
	}

	ok = len(res) > 0
	return
}

type RoutesCache map[string]map[string][]RouterResponse // map[path][method][statuscode][]RouterResponse