package http

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/AppsFlyer/go-sundheit"
	"github.com/AppsFlyer/go-sundheit/checks"
	"github.com/stretchr/testify/assert"
)

func TestHandleHealthJSON_longFormatNoChecks(t *testing.T) {
	h := health.New()
	resp := execReq(h, true)
	body, _ := ioutil.ReadAll(resp.Body)

	assert.Equal(t, http.StatusOK, resp.StatusCode, "status when no checks are registered")
	assert.Equal(t, "{}\n", string(body), "body when no checks are registered")
}

func TestHandleHealthJSON_shortFormatNoChecks(t *testing.T) {
	h := health.New()
	resp := execReq(h, false)
	body, _ := ioutil.ReadAll(resp.Body)

	assert.Equal(t, http.StatusOK, resp.StatusCode, "status when no checks are registered")
	assert.Equal(t, "{}\n", string(body), "body when no checks are registered")
}

func TestHandleHealthJSON_longFormatPassingCheck(t *testing.T) {
	h := health.New()

	err := h.RegisterCheck(createCheck("check1", true, 10*time.Millisecond))
	if err != nil {
		t.Error("Failed to register check: ", err)
	}
	defer h.DeregisterAll()

	resp := execReq(h, true)
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode, "status before first run")

	var respMsg = unmarshalLongFormat(resp.Body)
	const freshCheckMsg = "didn't run yet"
	expectedResponse := response{
		Check1: checkResult{
			Message: freshCheckMsg,
			Error: Err{
				Message: freshCheckMsg,
			},
			ContiguousFailures: 1,
		},
	}
	assert.Equal(t, &expectedResponse, respMsg, "body when no checks are registered")

	time.Sleep(11 * time.Millisecond)
	resp = execReq(h, true)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "status before first run")

	respMsg = unmarshalLongFormat(resp.Body)
	expectedResponse = response{
		Check1: checkResult{
			Message:            "pass",
			ContiguousFailures: 0,
		},
	}
	assert.Equal(t, &expectedResponse, respMsg, "body after first run")
}

func TestHandleHealthJSON_shortFormatPassingCheck(t *testing.T) {
	h := health.New()

	err := h.RegisterCheck(createCheck("check1", true, 10*time.Millisecond))
	if err != nil {
		t.Error("Failed to register check: ", err)
	}
	defer h.DeregisterAll()

	resp := execReq(h, false)
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode, "status before first run")

	var respMsg = unmarshalShortFormat(resp.Body)
	expectedResponse := map[string]string{"check1": "FAIL"}
	assert.Equal(t, expectedResponse, respMsg, "body when no checks are registered")

	time.Sleep(11 * time.Millisecond)
	resp = execReq(h, false)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "status before first run")

	respMsg = unmarshalShortFormat(resp.Body)
	expectedResponse = map[string]string{"check1": "PASS"}
	assert.Equal(t, expectedResponse, respMsg, "body after first run")
}

func unmarshalShortFormat(r io.Reader) map[string]string {
	respMsg := make(map[string]string)
	_ = json.NewDecoder(r).Decode(&respMsg)
	return respMsg
}

func unmarshalLongFormat(r io.Reader) *response {
	var respMsg response
	_ = json.NewDecoder(r).Decode(&respMsg)
	return &respMsg
}

func createCheck(name string, passing bool, delay time.Duration) *health.Config {
	return &health.Config{
		InitialDelay:    delay,
		ExecutionPeriod: delay,
		Check: &checks.CustomCheck{
			CheckName: name,
			CheckFunc: func() (details interface{}, err error) {
				if passing {
					return "pass", nil
				}
				return "failing", fmt.Errorf("failing")
			},
		},
	}
}

func execReq(h health.Health, longFormat bool) *http.Response {
	var path = "/meh"
	if !longFormat {
		path = fmt.Sprintf("%s?type=%s", path, ReportTypeShort)
	}

	handler := HandleHealthJSON(h)

	req := httptest.NewRequest(http.MethodGet, path, nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
	return w.Result()
}

type response struct {
	Check1 checkResult `json:"check1"`
}

type checkResult struct {
	Message            string `json:"message"`
	Error              Err    `json:"error"`
	ContiguousFailures int64  `json:"contiguousFailures"`
}

type Err struct {
	Message string `json:"message"`
}
