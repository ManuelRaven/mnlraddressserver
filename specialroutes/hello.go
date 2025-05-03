package specialroutes

import (
	"fmt"
	"net/http"
)

func Hellohandler(w http.ResponseWriter, r *http.Request) {

	if r.URL.Query().Get("name") == "" {
		sendJSONResponse(w, UploadResponse{
			Success: false,
			Message: "Missing name parameter",
		})

		return
	}

	fmt.Fprintf(w, "ping %s\n", r.URL.Query().Get("name"))

}
