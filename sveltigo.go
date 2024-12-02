package sveltigo

import (
	"context"
	"io/fs"
	"net/http"
	"strings"

	"github.com/HazelnutParadise/sveltigo/render"
)

// Props is an alias for map[string]any. It exists for documentation purposes.
// Props must be JSON-serializable when passing to fuctions defined in this package.
// ===
// Props 是 map[string]any 的別名，主要用於文檔說明目的。
// 當傳遞給此包中定義的函數時，Props 必須是可 JSON 序列化的。
type Props = map[string]any

// New constructs a golte middleware from the given filesystem.
// The root of the filesystem should be the golte build directory.
//
// The returned middleware is used to add a render context to incoming requests.
// It will allow you to use [Layout], [AddLayout], [Page], and [RenderPage].
// It should be mounted on the root of your router.
// The middleware should not be mounted on routes other than the root.
// ===
// New 從給定的文件系統構建一個 golte 中間件。
// 文件系統的根目錄應該是 golte 的構建目錄。
//
// 返回的中間件用於為傳入請求添加渲染上下文。
// 它允許你使用 [Layout]、[AddLayout]、[Page] 和 [RenderPage]。
// 它應該掛載在路由器的根目錄上。
// 此中間件不應該掛載在根目錄以外的路由上。
func New(fsys fs.FS) func(http.Handler) http.Handler {
	serverDir, err := fs.Sub(fsys, "server")
	if err != nil {
		panic(err)
	}

	clientDir, err := fs.Sub(fsys, "client")
	if err != nil {
		panic(err)
	}

	renderer := render.New(&serverDir, &clientDir)
	assets := http.StripPrefix("/"+renderer.Assets()+"/", fileServer(clientDir))
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/"+renderer.Assets()+"/") {
				assets.ServeHTTP(w, r)
				return
			}

			scheme := "http"
			if r.TLS != nil {
				scheme += "s"
			}

			ctx := context.WithValue(r.Context(), contextKey{}, &RenderContext{
				Renderer: renderer,
				ErrPage:  "$$$GOLTE_DEFAULT_ERROR$$$",
				scdata: render.SvelteContextData{
					URL: scheme + "://" + r.Host + r.URL.String(),
				},
			})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Layout returns a middleware that calls [AddLayout].
// Use this when there are no props needed to render the component.
// If you need to pass props, use [AddLayout] instead.
// ===
// Layout 返回一個調用 [AddLayout] 的中間件。
// 當不需要 props 來渲染組件時使用此函數。
// 如果需要傳遞 props，請使用 [AddLayout]。
func Layout(component string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			AddLayout(r, component, nil)
			next.ServeHTTP(w, r)
		})
	}
}

// Error returns a middleware that calls [SetError].
// Use this when there are no props needed to render the component.
// If you need to pass props, use [SetError] instead.
// ===
// Error 返回一個調用 [SetError] 的中間件。
// 當不需要 props 來渲染組件時使用此函數。
// 如果需要傳遞 props，請使用 [SetError]。
func Error(component string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			SetError(r, component)
			next.ServeHTTP(w, r)
		})
	}
}

// Page returns a handler that calls [RenderPage].
// Use this when there are no props needed to render the component.
// If you need to pass props, use [RenderPage] instead.
// ===
// Page 返回一個調用 [RenderPage] 的處理器。
// 當不需要 props 來渲染組件時使用此函數。
// 如果需要傳遞 props，請使用 [RenderPage]。
func Page(component string) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		RenderPage(w, r, component, nil)
	})
}

// AddLayout appends the component to the request.
// Layouts consist of any components with a <slot>.
// Calling this multiple times on the same request will nest layouts.
// ===
// AddLayout 將組件附加到請求中。
// 佈局由包含 <slot> 的任何組件組成。
// 在同一請求上多次調用此函數將嵌套佈局。
func AddLayout(r *http.Request, component string, props Props) {
	rctx := MustGetRenderContext(r)
	rctx.Components = append(rctx.Components, render.Entry{
		Comp:  component,
		Props: props,
	})
}

// SetError sets the error page for the request.
// Errors consist of any components that take the "message" and "status" props.
// Calling this multiple times on the same request will overrite the previous error page.
// ===
// SetError 為請求設置錯誤頁面。
// 錯誤頁面由接受 "message" 和 "status" props 的任何組件組成。
// 在同一請求上多次調用此函數將覆蓋之前的錯誤頁面。
func SetError(r *http.Request, component string) {
	MustGetRenderContext(r).ErrPage = component
}

// RenderPage renders the specified component.
// If any layouts were added previously, then each subsequent layout will
// go in the <slot> of the previous layout. The page will be in the <slot>
// of the last layout.
// ===
// RenderPage 渲染指定的組件。
// 如果之前添加了任何佈局，那麼每個後續佈局將
// 放在前一個佈局的 <slot> 中。頁面將放在最後一個佈局的 <slot> 中。
func RenderPage(w http.ResponseWriter, r *http.Request, component string, props Props) {
	rctx := MustGetRenderContext(r)
	rctx.Components = append(rctx.Components, render.Entry{
		Comp:  component,
		Props: props,
	})
	rctx.Render(w)
}

// RenderError renders the current error page along with layouts.
// The error componenet will receive "message" and "status" as props.
// It will also write the status code to the header.
// ===
// RenderError 渲染當前錯誤頁面及其佈局。
// 錯誤組件將接收 "message" 和 "status" 作為 props。
// 它還會將狀態碼寫入標頭。
func RenderError(w http.ResponseWriter, r *http.Request, message string, status int) {
	rctx := MustGetRenderContext(r)
	entry := render.Entry{Comp: rctx.ErrPage, Props: Props{
		"message": message,
		"status":  status,
	}}
	rctx.Components = append(rctx.Components, entry)
	rctx.Render(respWriterWrapper{w})
}

// respWriterWrapper is needed to prevent superfluous WriteHeader calls
// ===
// respWriterWrapper 用於防止多餘的 WriteHeader 調用
type respWriterWrapper struct {
	http.ResponseWriter
}

func (w respWriterWrapper) WriteHeader(status int) {
	w.ResponseWriter.WriteHeader(http.StatusInternalServerError)
}
