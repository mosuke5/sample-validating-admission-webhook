package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func main() {
	var (
		serverCert = flag.String("server-cert", "./server.crt", "Server certificate")
		serverKey  = flag.String("server-key", "./server.key", "Server key")
		serverPort = flag.String("port", "8443", "Server listen port")
		bodyDump   = flag.Bool("body-dump", false, "Enable body dump")
	)

	flag.Parse()
	e := echo.New()

	// for debug
	if *bodyDump {
		e.Use(middleware.BodyDump(func(c echo.Context, reqBody, resBody []byte) {
			fmt.Fprintf(os.Stderr, "Request body: %v\nResponse body: %v\n", string(reqBody), string(resBody))
		}))
	}

	e.POST("/runasuser-validation", runAsUserValidation)

	s := http.Server{
		Addr:    ":" + *serverPort,
		Handler: e,
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}

	if err := s.ListenAndServeTLS(*serverCert, *serverKey); err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

// admin namespace => any runAsUser is ok
// user namespace  => runAsUser other than root or empty is ok
func runAsUserValidation(c echo.Context) error {
	req := new(admissionv1.AdmissionReview)
	res := new(admissionv1.AdmissionReview)
	res.Response = new(admissionv1.AdmissionResponse)

	if err := c.Bind(req); err != nil {
		panic(err)
	}

	// Fetch pod data
	var pod corev1.Pod
	if err := json.Unmarshal(req.Request.Object.Raw, &pod); err != nil {
		panic(err)
	}

	if isAdminNamespace(req.Request.Namespace) {
		res.Response.Allowed = true
		return returnResponse(req, res, c)
	}

	if pod.Spec.SecurityContext.RunAsUser == nil {
		res.Response.Allowed = false
		res.Response.Result = &metav1.Status{
			Code:    http.StatusForbidden,
			Message: "runAsUser is required in user namespace.",
		}
		return returnResponse(req, res, c)
	}

	if isRootUser(pod.Spec.SecurityContext.RunAsUser) {
		res.Response.Allowed = false
		res.Response.Result = &metav1.Status{
			Code:    http.StatusForbidden,
			Message: "Can't set root for runAsUser in user namespace.",
		}
		return returnResponse(req, res, c)

	} else {
		res.Response.Allowed = true
		return returnResponse(req, res, c)
	}
}

func isAdminNamespace(ns string) bool {
	reg := `^admin-*`
	if regexp.MustCompile(reg).Match([]byte(ns)) {
		return true
	} else {
		return false
	}
}

func isRootUser(uid *int64) bool {
	var root int64 = 0
	if *uid == root {
		return true
	} else {
		return false
	}
}

func returnResponse(request *admissionv1.AdmissionReview, response *admissionv1.AdmissionReview, c echo.Context) error {
	response.Kind = request.Kind
	response.APIVersion = request.APIVersion
	response.Response.UID = request.Request.UID
	return c.JSON(http.StatusOK, response)
}
