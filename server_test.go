package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	admissionv1 "k8s.io/api/admission/v1"
)

func readRequestTemplate(file string) admissionv1.AdmissionReview {
	f, err := os.Open(file)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	var reqAr admissionv1.AdmissionReview
	if err := json.NewDecoder(bufio.NewReader(f)).Decode(&reqAr); err != nil {
		panic(err)
	}
	return reqAr
}

func postRequest(e *echo.Echo, reqAr admissionv1.AdmissionReview) (*httptest.ResponseRecorder, admissionv1.AdmissionReview) {
	b, err := json.Marshal(reqAr)
	if err != nil {
		panic(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/runasuser-validation", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	c := e.NewContext(req, rec)

	if err := runAsUserValidation(c); err != nil {
		panic(err)
	}

	var resp admissionv1.AdmissionReview
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		panic(err)
	}

	return rec, resp
}

// runAsUser is nonRoot, namespace is user
// => Allowed is true
func TestRunAsNonRootInUserNamespace(t *testing.T) {
	e := echo.New()

	//必要なパラメータセット
	reqAr := readRequestTemplate("testdata/nonRootRequestTemplate.json")
	reqAr.Request.Namespace = "user-namespace"

	rec, resp := postRequest(e, reqAr)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, reqAr.Request.UID, resp.Response.UID)
	assert.Equal(t, true, resp.Response.Allowed)
}

// runAsUser is root, namespace is user
// => Allowed is false
func TestRunAsRootInUserNamespace(t *testing.T) {
	e := echo.New()

	//必要なパラメータセット
	reqAr := readRequestTemplate("testdata/rootRequestTemplate.json")
	reqAr.Request.Namespace = "user-namespace"

	rec, resp := postRequest(e, reqAr)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, int32(http.StatusForbidden), resp.Response.Result.Code)
	assert.Equal(t, "Can't set root for runAsUser in user namespace.", resp.Response.Result.Message)
	assert.Equal(t, reqAr.Request.UID, resp.Response.UID)
	assert.Equal(t, false, resp.Response.Allowed)
}

// runAsUser is root, namespace is admin
// => Allowed is true
func TestRunAsRootInAdminNamespace(t *testing.T) {
	e := echo.New()

	//必要なパラメータセット
	reqAr := readRequestTemplate("testdata/nonRootRequestTemplate.json")
	reqAr.Request.Namespace = "admin-namespace"

	rec, resp := postRequest(e, reqAr)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, reqAr.Request.UID, resp.Response.UID)
	assert.Equal(t, true, resp.Response.Allowed)
}

// runAsUser is emty, namespace is user
// => Allowed is false
func TestNoRunAsUserInUserNamespace(t *testing.T) {
	e := echo.New()

	//必要なパラメータセット
	reqAr := readRequestTemplate("testdata/noRunAsUserRequestTemplate.json")
	reqAr.Request.Namespace = "user-namespace"

	rec, resp := postRequest(e, reqAr)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, int32(http.StatusForbidden), resp.Response.Result.Code)
	assert.Equal(t, "runAsUser is required in user namespace.", resp.Response.Result.Message)
	assert.Equal(t, reqAr.Request.UID, resp.Response.UID)
	assert.Equal(t, false, resp.Response.Allowed)
}

// runAsUser is emty, namespace is admin
// => Allowed is true
func TestNoRunAsUserInAdminNamespace(t *testing.T) {
	e := echo.New()

	//必要なパラメータセット
	reqAr := readRequestTemplate("testdata/noRunAsUserRequestTemplate.json")
	reqAr.Request.Namespace = "admin-namespace"

	rec, resp := postRequest(e, reqAr)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, reqAr.Request.UID, resp.Response.UID)
	assert.Equal(t, true, resp.Response.Allowed)
}
