package main

import (
	"fmt"
	"net/http"
)

func main() {
	// same app using different routers
	// comment/uncomment to try the different routers.

	r := chiRouter()
	// r := serveMuxRouter()
	// r := gorillaMuxRouter()
	// r := echoRouter()
	// r := ginRouter()

	fmt.Println("Serving on :8000")
	http.ListenAndServe(":8000", r)
}
