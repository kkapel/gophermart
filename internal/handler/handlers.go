package handler

import "net/http"

type Handler struct {
}

func (h *Handler) RegisterUser(res http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodPost:
	default:
		errorResponse(res)
	}

}

func errorResponse(res http.ResponseWriter) {
	res.WriteHeader(http.StatusBadRequest)
}
