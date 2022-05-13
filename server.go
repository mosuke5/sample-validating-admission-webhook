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

	res.UID = req.Request.UID

	// Pod情報取り出し
	var pod corev1.Pod
	if err := json.Unmarshal(req.Request.Object.Raw, &pod); err != nil {
		panic(err)
	}

	// RunAsUseerが空の場合は拒否
	if pod.Spec.SecurityContext.RunAsUser == nil {
		res.Allowed = false
		return returnResponse(req.APIVersion, req.Kind, res, c)
	}

	// runasuserがroot以外なら許可して終了
	if !isRootUser(pod.Spec.SecurityContext.RunAsUser) {
		res.Allowed = true
		return returnResponse(req.APIVersion, req.Kind, res, c)
	}

	// runasuserがrootの場合はnamespace名で判断する
	if isAdminNamespace(req.Request.Namespace) {
		res.Allowed = true
		return returnResponse(req.APIVersion, req.Kind, res, c)
	}

	res.Allowed = false
	return returnResponse(req.APIVersion, req.Kind, res, c)
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

func returnResponse(apiVersion string, kind string, response *admissionv1.AdmissionResponse, c echo.Context) error {
	return c.JSON(http.StatusOK, Response{
		ApiVersion: apiVersion,
		Kind:       kind,
		Response:   response,
	})
}
