package sveltigo_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/HazelnutParadise/sveltigo"
	"github.com/HazelnutParadise/sveltigo/testdata"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

var (
	server  *httptest.Server
	browser *rod.Browser
)

func TestMain(m *testing.M) {
	mux := http.NewServeMux()

	e0 := sveltigo.Error("error/0")
	e1 := sveltigo.Error("error/1")
	l0 := sveltigo.Layout("layout/0")
	l1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sveltigo.AddLayout(r, "layout/1", map[string]any{"varNum": 69, "varStr": "mystring", "varBool": true})
			next.ServeHTTP(w, r)
		})
	}
	l2 := sveltigo.Layout("layout/2")
	l3 := sveltigo.Layout("layout/3")
	p0 := sveltigo.Page("page/0")
	p1 := sveltigo.Page("page/1")
	p2 := sveltigo.Page("page/2")
	p3 := sveltigo.Page("page/3")

	mux.Handle("/route0", e0(l0(l1(l2(p0)))))
	mux.Handle("/route1", e0(l0(l1(l2(p1)))))
	mux.Handle("/route2", e0(l0(l1(l2(p2)))))
	mux.Handle("/route3", e0(l0(l1(l2(p3)))))
	mux.Handle("/error0", e0(l0(l3(p1))))
	mux.Handle("/error1", e1(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sveltigo.RenderError(w, r, "mymessage", 401)
	})))

	server = httptest.NewServer(sveltigo.New(testdata.App)(mux))
	defer server.Close()

	browser = rod.New()
	defer browser.Close()

	err := browser.Connect()
	if err != nil {
		panic(err)
	}

	os.Exit(m.Run())
}

func checkProps(el *rod.Element, propMap map[string]any) func(*testing.T) {
	return func(t *testing.T) {
		ch := make(chan *rod.Element)
		go func() {
			ch <- el.MustElement("#props")
		}()

		select {
		case <-time.After(time.Second):
			t.Fatal("#props not found")
		case props := <-ch:
			for k, v := range propMap {
				prop := props.MustAttribute(k)
				if prop == nil {
					t.Errorf("%s is not defined", k)
				} else if *prop != fmt.Sprintf("%v", v) {
					t.Errorf("expected %v for %s, instead got %s", v, k, *prop)
				}
			}
		}
	}
}

func doCommonChecks(t *testing.T, page *rod.Page) {
	el := page.MustElement("#layout1")
	t.Run("check props", checkProps(el, map[string]any{
		"varNum":  69,
		"varStr":  "mystring",
		"varBool": true,
	}))

	t.Run("check style", func(t *testing.T) {
		o := page.MustEval("() => getComputedStyle(document.querySelector('#layout0 > #layout1 > #layout2 > #page0'))['font-size']")
		if o.Str() != "30px" {
			t.Fail()
		}
	})
}

func TestJSDisabled(t *testing.T) {
	err := rod.Try(func() {
		page := browser.MustPage("")
		router := page.HijackRequests()
		router.MustAdd("*.js", func(ctx *rod.Hijack) { ctx.Response.Fail(proto.NetworkErrorReasonBlockedByClient) })
		go router.Run()
		page.MustNavigate(server.URL + "/route0")
		page.MustWaitLoad()
		if !page.MustHas("#layout0 > #layout1 > #layout2 > #page0 > #ssr") {
			t.FailNow()
		}

		doCommonChecks(t, page)
	})
	if err != nil {
		t.Error(err)
	}
}

func TestJSEnabled(t *testing.T) {
	err := rod.Try(func() {
		page := browser.MustPage("")

		r1 := make(chan struct{})
		r2 := make(chan struct{})
		r3 := make(chan struct{})

		router := page.HijackRequests()
		router.MustAdd("*", func(ctx *rod.Hijack) {
			var ch chan struct{}
			switch ctx.Request.URL().Path {
			case "/route1":
				ch = r1
			case "/route2":
				ch = r2
			case "/route3":
				ch = r3
			default:
				ctx.ContinueRequest(&proto.FetchContinueRequest{})
				return
			}

			if ctx.Request.IsNavigation() {
				ctx.Response.Fail(proto.NetworkErrorReasonBlockedByClient)
			} else {
				close(ch)
				ctx.ContinueRequest(&proto.FetchContinueRequest{})
			}
		})

		go router.Run()

		page.MustNavigate(server.URL + "/route0")
		page.MustWaitLoad()
		page.MustWaitStable()
		if !page.MustHas("#layout0 > #layout1 > #layout2 > #page0 > #csr") {
			t.FailNow()
		}

		doCommonChecks(t, page)

		t.Run("check noreload", func(t *testing.T) {
			if !page.MustHas("#layout2 > a[href='/route1']") || !page.MustHas("#layout2 > a[href='/route2']") || !page.MustHas("#layout2 > a[href='/route3']") {
				t.Fatal("elements not found")
			}

			mount := page.MustElement("#layout2 > a[href='/route1']")
			hover := page.MustElement("#layout2 > a[href='/route2']")
			tap := page.MustElement("#layout2 > a[href='/route3']")

			select {
			case <-r1:
			case <-time.After(time.Second):
				t.Error("mount link not loading on mount")
			}

			hover.MustHover()
			select {
			case <-r2:
			case <-time.After(time.Second):
				t.Error("hover link not loading on hover")
			}

			mount.MustClick()
			page.MustWaitStable()
			if !page.MustHas("#layout0 > #layout1 > #layout2 > #page1") {
				t.Fatal("route0 not working")
			}

			hover.MustClick()
			page.MustWaitStable()
			if !page.MustHas("#layout0 > #layout1 > #layout2 > #page2") {
				t.Fatal("route0 not working")
			}

			tap.MustClick()
			page.MustWaitStable()
			if !page.MustHas("#layout0 > #layout1 > #layout2 > #page3") {
				t.Fatal("route0 not working")
			}

			t.Run("check history navigation", func(t *testing.T) {
				page.MustNavigateBack()
				page.MustWaitStable()
				if !page.MustHas("#layout0 > #layout1 > #layout2 > #page2") {
					t.Fatal("could not nav back")
				}

				page.MustNavigateBack()
				page.MustWaitStable()
				if !page.MustHas("#layout0 > #layout1 > #layout2 > #page1") {
					t.Fatal("could not nav back")
				}

				page.MustNavigateBack()
				page.MustWaitStable()
				if !page.MustHas("#layout0 > #layout1 > #layout2 > #page0") {
					t.Log(page.MustHTML())
					t.Fatal("could not nav back")
				}

				page.MustNavigateForward()
				page.MustWaitStable()
				if !page.MustHas("#layout0 > #layout1 > #layout2 > #page1") {
					t.Log(page.MustHTML())
					t.Fatal("could not nav forward")
				}

				page.MustNavigateForward()
				page.MustWaitStable()
				if !page.MustHas("#layout0 > #layout1 > #layout2 > #page2") {
					t.Log(page.MustHTML())
					t.Fatal("could not nav forward")
				}

				page.MustNavigateForward()
				page.MustWaitStable()
				if !page.MustHas("#layout0 > #layout1 > #layout2 > #page3") {
					t.Log(page.MustHTML())
					t.Fatal("could not nav forward")
				}
			})
		})
	})
	if err != nil {
		t.Error(err)
	}
}

func TestErrorPage(t *testing.T) {
	err := rod.Try(func() {
		t.Run("check render error", func(t *testing.T) {
			page := browser.MustPage(server.URL + "/error0")
			page.MustWaitLoad()
			page.MustWaitStable()

			if !page.MustHas("#error0") {
				t.FailNow()
			}
			el := page.MustElement("#error0")

			t.Run("check props", checkProps(el, map[string]any{
				"status": 500,
			}))

			t.Run("check index", func(t *testing.T) {
				if !page.MustHas("#layout0 > #error0") {
					t.Fail()
				}

				if page.MustHas("#layout3") || page.MustHas("#page1") {
					t.Fail()
				}
			})
		})

		t.Run("check manual error", func(t *testing.T) {
			page := browser.MustPage(server.URL + "/error1")
			page.MustWaitLoad()
			page.MustWaitStable()

			if !page.MustHas("#error1") {
				t.FailNow()
			}
			el := page.MustElement("#error1")

			t.Run("check props", checkProps(el, map[string]any{
				"status":  401,
				"message": "mymessage",
			}))
		})
	})
	if err != nil {
		t.Error(err)
	}
}
