package integrated

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/channel-io/cht-app-github/api/public"
	libhttp "github.com/channel-io/cht-app-github/internal/http"
	"github.com/channel-io/cht-app-github/tool"
)

func TestPing(t *testing.T) {
	tool.NewIntegratedTestSuite().
		Target(public.HTTPServerTestModule()).
		Target(integratedTestModule()).
		Test(func(server *libhttp.Server, w *httptest.ResponseRecorder) {
			req, _ := http.NewRequest("GET", "/ping", nil)
			server.Serve(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "pong", w.Body.String())
		}).
		Run()
}
