package front

import "net/http"

func FrontendHandler() http.Handler {
	return http.FileServer(http.Dir("./static"))
}
