package golte_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/nichady/golte"
	"github.com/nichady/golte/testdata"
)

var (
	server  *httptest.Server
	browser *rod.Browser
)

func TestMain(m *testing.M) {
	middleware, assets := golte.From(testdata.App)

	mux := http.NewServeMux()
	mux.Handle("/app_/", assets)

	e0 := golte.Error("error/0")
	e1 := golte.Error("error/1")
	l0 := golte.Layout("layout/0")
	l1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			golte.AddLayout(r, "layout/1", map[string]any{"varNum": 69, "varStr": "mystring", "varBool": true})
			next.ServeHTTP(w, r)
		})
	}
	l2 := golte.Layout("layout/2")
	l3 := golte.Layout("layout/3")
	p0 := golte.Page("page/0")
	p1 := golte.Page("page/1")
	p2 := golte.Page("page/2")
	p3 := golte.Page("page/3")

	mux.Handle("/route0", e0(l0(l1(l2(p0)))))
	mux.Handle("/route1", e0(l0(l1(l2(p1)))))
	mux.Handle("/route2", e0(l0(l1(l2(p2)))))
	mux.Handle("/route3", e0(l0(l1(l2(p3)))))
	mux.Handle("/error0", e0(l0(l3(p1))))
	mux.Handle("/error1", e1(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		golte.RenderError(w, r, "mymessage", 401)
	})))

	server = httptest.NewServer(middleware(mux))
	defer server.Close()

	browser = rod.New()
	defer browser.Close()

	err := browser.Connect()
	if err != nil {
		panic(err)
	}

	os.Exit(m.Run())
}

func doCommonChecks(t *testing.T, page *rod.Page) {
	t.Run("check props", func(t *testing.T) {
		if !page.MustHas("#layout1 > #varNum") || !page.MustHas("#layout1 > #varStr") || !page.MustHas("#layout1 > #varBool") {
			t.Fatal("elements not found")
		}

		varNum := *page.MustElement("#layout1 > #varNum").MustAttribute("val")
		if varNum != "69" {
			t.Errorf("varNum unexpected value: %s", varNum)
		}

		varStr := *page.MustElement("#layout1 > #varStr").MustAttribute("val")
		if varStr != "mystring" {
			t.Errorf("varStr unexpected value: %s", varStr)
		}

		varBool := *page.MustElement("#layout1 > #varBool").MustAttribute("val")
		if varBool != "true" {
			t.Errorf("varBool unexpected value: %s", varBool)
		}
	})

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
			if !page.MustHas("#layout2 > a[noreload='mount']") || !page.MustHas("#layout2 > a[noreload='hover']") || !page.MustHas("#layout2 > a[noreload='tap']") {
				t.Fatal("elements not found")
			}

			mount := page.MustElement("#layout2 > a[noreload='mount']")
			hover := page.MustElement("#layout2 > a[noreload='hover']")
			tap := page.MustElement("#layout2 > a[noreload='tap']")

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

			t.Run("check status code", func(t *testing.T) {
				if !page.MustHas("#status") {
					t.FailNow()
				}
				status := *page.MustElement("#status").MustAttribute("status")
				if status != "500" {
					t.Fail()
				}
			})

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

			t.Run("check status code", func(t *testing.T) {
				if !page.MustHas("#status") {
					t.Fatal("status not found")
				}
				status := page.MustElement("#status").MustAttribute("status")
				if status == nil {
					t.Fatal("status is undefined")
				}
				if *status != "401" {
					t.Fatalf("unexpected status: %s", *status)
				}
			})

			t.Run("check message", func(t *testing.T) {
				if !page.MustHas("#message") {
					t.FailNow()
				}
				message := page.MustElement("#message").MustAttribute("message")
				if message == nil {
					t.FailNow()
				}
				if *message != "mymessage" {
					t.Fail()
				}
			})
		})
	})
	if err != nil {
		t.Error(err)
	}
}
