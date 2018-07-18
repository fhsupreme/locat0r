package main

import (
	"bytes"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/labstack/echo"
)

var healthTmpl = template.New("health")

func main() {
	healthTmpl.Parse(`
	<code>
		Protocol: {{.Proto}}<br>
		Host: {{.Host}}<br>
		Remote Address: {{.RemoteAddr}}<br>
		Method: {{.Method}}<br>
		Path: {{.URL.Path}}<br>
	</code>
	`)

	e := echo.New()
	e.Static("/", "static")
	e.GET("/health", health)
	e.POST("/position", postPosition)

	e.Logger.Fatal(e.StartTLS(":8023", "cert.pem", "key.pem"))
}

func health(c echo.Context) error {
	var data bytes.Buffer
	healthTmpl.Execute(&data, c.Request())
	return c.HTML(http.StatusOK, data.String())
}

func postPosition(c echo.Context) error {
	data := make(map[string]interface{})
	if err := c.Bind(&data); err != nil {
		log.Printf("postPosition error: %v\n", err)
	} else {
		log.Printf("Got position: %v\n", data)
	}
	data["t"] = time.Now()
	return c.JSON(http.StatusOK, data)
}
