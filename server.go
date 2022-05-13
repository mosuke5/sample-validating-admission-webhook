package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"regexp"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
)

type Response struct {
	ApiVersion string                         `json:"apiVersion"`
	Kind       string                         `json:"kind"`
	Response   *admissionv1.AdmissionResponse `json:"response"`
}

func main() {
	var (
		serverCert = flag.String("server-cert", "./server.crt", "Server certificate")
		serverKey  = flag.String("server-key", "./server.key", "Server key")
		serverPort = flag.String("port", "8443", "Server listen port")
	)
	flag.Parse()
	e := echo.New()

	// 練習のためリクエストボディをログに吐き出しておきます。
	// 実際に来たリクエストを確認するためです。
	e.Use(middleware.BodyDump(func(c echo.Context, reqBody, resBody []byte) {
		//fmt.Fprintf(os.Stderr, "Request body: %v\n", string(reqBody))
	}))

	e.POST("/runasuser-validation", runAsUserValidation)

	s := http.Server{
		Addr:      ":" + *serverPort,
		Handler:   e,
		TLSConfig: &tls.Config{
			//MinVersion: 1, // customize TLS configuration
		},
		//ReadTimeout: 30 * time.Second, // use custom timeouts
	}
	if err := s.ListenAndServeTLS(*serverCert, *serverKey); err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func runAsUserValidation(c echo.Context) error {
	req := new(admissionv1.AdmissionReview)
	res := new(admissionv1.AdmissionResponse)

	if err := c.Bind(req); err != nil {
		panic(err)
	}

	var root int64 = 0

	// Pod情報取り出し
	var pod corev1.Pod
	if err := json.Unmarshal(req.Request.Object.Raw, &pod); err != nil {
		panic(err)
	}

	// RunAsUseerが空の場合は拒否
	if pod.Spec.SecurityContext.RunAsUser == nil {
		res.UID = req.Request.UID
		res.Allowed = false
		return c.JSON(http.StatusForbidden, Response{
			ApiVersion: req.APIVersion,
			Kind:       req.Kind,
			Response:   res,
		})
	}

	// runasuserがroot以外なら許可して終了
	if *pod.Spec.SecurityContext.RunAsUser != root {
		res.UID = req.Request.UID
		res.Allowed = true

		return c.JSON(http.StatusOK, Response{
			ApiVersion: req.APIVersion,
			Kind:       req.Kind,
			Response:   res,
		})
	}

	// runasuserがrootの場合はnamespace名で判断する
	if isAdminNamespace(req.Request.Namespace) {
		res.UID = req.Request.UID
		res.Allowed = true
		return c.JSON(http.StatusOK, Response{
			ApiVersion: req.APIVersion,
			Kind:       req.Kind,
			Response:   res,
		})
	}

	res.UID = req.Request.UID
	res.Allowed = false
	return c.JSON(http.StatusForbidden, Response{
		ApiVersion: req.APIVersion,
		Kind:       req.Kind,
		Response:   res,
	})
}

func isAdminNamespace(ns string) bool {
	reg := `^admin-*`
	if regexp.MustCompile(reg).Match([]byte(ns)) {
		return true
	} else {
		return false
	}
}

func returnResponse(apiVersion string, kind string, response *admissionv1.AdmissionResponse, c echo.Context) error {
	return c.JSON(http.StatusForbidden, Response{
		ApiVersion: apiVersion,
		Kind:       kind,
		Response:   response,
	})
}
