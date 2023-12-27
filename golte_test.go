package golte_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

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
	middleware, assets := golte.From(testdata.App, golte.Options{})

	mux := http.NewServeMux()
	mux.Handle("/app_/", assets)

	e0 := golte.Error("error/0")
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

	mux.Handle("/route0", e0(l0(l1(l2(p0)))))
	mux.Handle("/route1", e0(l0(l3(p1))))

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
		if !page.MustHas("#layout1-varNum") || !page.MustHas("#layout1-varStr") || !page.MustHas("#layout1-varBool") {
			t.Fatal("elements not found")
		}

		varNum := *page.MustElement("#layout1-varNum").MustAttribute("val")
		if varNum != "69" {
			t.Errorf("varNum unexpected value: %s", varNum)
		}

		varStr := *page.MustElement("#layout1-varStr").MustAttribute("val")
		if varStr != "mystring" {
			t.Errorf("varStr unexpected value: %s", varStr)
		}

		varBool := *page.MustElement("#layout1-varBool").MustAttribute("val")
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
		if !page.MustHas("#layout0 > #layout1 > #layout2 > #page0-ssr") {
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
		page := browser.MustPage(server.URL + "/route0")
		page.MustWaitLoad()
		page.MustWaitStable()
		if !page.MustHas("#layout0 > #layout1 > #layout2 > #page0-csr") {
			t.FailNow()
		}

		doCommonChecks(t, page)
	})
	if err != nil {
		t.Error(err)
	}
}

func TestErrorPage(t *testing.T) {
	err := rod.Try(func() {
		page := browser.MustPage(server.URL + "/route1")
		page.MustWaitLoad()
		page.MustWaitStable()
		if !page.MustHas("#error0") {
			t.FailNow()
		}

		t.Run("check index", func(t *testing.T) {
			if !page.MustHas("#layout0 > #error0") {
				t.Fail()
			}

			if page.MustHas("#layout3") || page.MustHas("#page1") {
				t.Fail()
			}
		})
	})
	if err != nil {
		t.Error(err)
	}
}
