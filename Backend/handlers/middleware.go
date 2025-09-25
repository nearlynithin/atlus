package handlers

import (
	"context"
	"log"
	"net/http"
)

func Authenticator(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		c, err := r.Cookie("session")
		if err != nil {
			http.Redirect(w, r, "/login/", http.StatusSeeOther)
			log.Print("session invalid ", http.StatusUnauthorized)
			return
		}

		sdata, err := getSessionData(ctx, c.Value)
		if err != nil {
			http.Redirect(w, r, "/login/", http.StatusSeeOther)
			log.Print("Please login to play ", http.StatusUnauthorized)
			return
		}

		ctx = context.WithValue(ctx, "sessionData", sdata)
		handler.ServeHTTP(w, r.WithContext(ctx))
	}
}
