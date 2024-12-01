package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"examples/blog/database"

	_ "modernc.org/sqlite"

	"github.com/HazelnutParadise/sveltigo"
	"github.com/go-chi/chi/v5"
)

var db = database.NewDB("data.db")

func main() {
	r := chi.NewRouter()

	r.Use(auth)
	r.Use(build.sveltigo)

	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sveltigo.AddLayout(r, "layout/main", map[string]any{"user": username(r)})
			next.ServeHTTP(w, r)
		})
	})

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/blog", http.StatusMovedPermanently)
	})

	blogRoutes(r)
	authRoutes(r)

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		sveltigo.RenderError(w, r, "Not Found", http.StatusNotFound)
	})

	fmt.Println("Serving on :8000")
	http.ListenAndServe(":8000", r)
}

func blogRoutes(r chi.Router) {
	r.Get("/blog", func(w http.ResponseWriter, r *http.Request) {
		blogs, err := db.GetAllBlogs()
		if err != nil {
			sveltigo.RenderError(w, r, err.Error(), http.StatusInternalServerError)
			return
		}

		sveltigo.RenderPage(w, r, "page/blogs", map[string]any{
			"blogs": blogs,
		})
	})

	r.Get("/blog/{id:[0-9+]}", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			sveltigo.RenderError(w, r, err.Error(), http.StatusInternalServerError)
			return
		}

		blog, err := db.GetBlog(id)
		if errors.Is(err, database.ErrBlogNotExist) {
			sveltigo.RenderError(w, r, err.Error(), http.StatusNotFound)
			return
		} else if err != nil {
			sveltigo.RenderError(w, r, err.Error(), http.StatusInternalServerError)
			return
		}

		sveltigo.RenderPage(w, r, "page/blog", map[string]any{
			"blog": blog,
		})
	})

	r.Post("/blog", func(w http.ResponseWriter, r *http.Request) {
		if username(r) == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		err := r.ParseForm()
		if err != nil {
			sveltigo.RenderError(w, r, err.Error(), http.StatusBadRequest)
			return
		}

		title := r.Form.Get("title")
		body := r.Form.Get("body")

		err = db.PostBlog(username(r), title, body)
		if err != nil {
			sveltigo.RenderError(w, r, err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/blog", http.StatusSeeOther)
	})

	r.Get("/new", func(w http.ResponseWriter, r *http.Request) {
		if username(r) == "" {
			sveltigo.RenderError(w, r, "Unauthorized", http.StatusUnauthorized)
			return
		}

		sveltigo.RenderPage(w, r, "page/new", nil)
	})

	r.Get("/user/{username}", func(w http.ResponseWriter, r *http.Request) {
		blogs, err := db.GetUserBlogs(chi.URLParam(r, "username"))

		if errors.Is(err, database.ErrUserNotExist) {
			sveltigo.RenderError(w, r, err.Error(), http.StatusNotFound)
			return
		} else if err != nil {
			sveltigo.RenderError(w, r, err.Error(), http.StatusInternalServerError)
			return
		}

		sveltigo.RenderPage(w, r, "page/userblogs", map[string]any{
			"blogs": blogs,
		})
	})
}

func authRoutes(r chi.Router) {
	r.Get("/login", sveltigo.Page("page/login"))

	r.Post("/login", func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			sveltigo.RenderError(w, r, err.Error(), http.StatusBadRequest)
			return
		}

		username := r.Form.Get("username")
		password := r.Form.Get("password")

		exists, err := db.AccountExists(username, password)
		if err != nil {
			sveltigo.RenderError(w, r, err.Error(), http.StatusInternalServerError)
			return
		}

		if !exists {
			http.Redirect(w, r, "/login?err=2", http.StatusSeeOther)
			return
		}

		id, err := db.CreateSession(username)
		if err != nil {
			sveltigo.RenderError(w, r, err.Error(), http.StatusInternalServerError)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:   "session",
			Value:  id,
			Secure: true,
		})

		http.Redirect(w, r, "/blog", http.StatusSeeOther)
	})

	r.Get("/logout", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name:   "session",
			MaxAge: -1,
		})
		http.Redirect(w, r, "/blog", http.StatusFound)
	})

	r.Post("/register", func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			sveltigo.RenderError(w, r, err.Error(), http.StatusBadRequest)
			return
		}

		username := r.Form.Get("username")
		password := r.Form.Get("password")

		err = db.RegisterAccount(username, password)
		if errors.Is(err, database.ErrAccountAlreadyExists) {
			http.Redirect(w, r, "/login?err=1", http.StatusSeeOther)
			return
		} else if err != nil {
			sveltigo.RenderError(w, r, err.Error(), http.StatusInternalServerError)
			return
		}

		id, err := db.CreateSession(username)
		if err != nil {
			sveltigo.RenderError(w, r, err.Error(), http.StatusInternalServerError)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:   "session",
			Value:  id,
			Secure: true,
		})

		http.Redirect(w, r, "/blog", http.StatusSeeOther)
	})
}

// auth is a middleware that adds username to the request context based on the session cookie.
func auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		username, err := db.GetSession(cookie.Value)
		if err != nil && !errors.Is(err, database.ErrSessionNotExist) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		ctx := context.WithValue(r.Context(), "username", username)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// username returns the username stored in the request context
func username(r *http.Request) string {
	username, _ := r.Context().Value("username").(string)
	return username
}
