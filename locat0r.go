package main

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo"
)

var healthTmpl = template.New("health")

var lastPosition position

type position struct {
	lon, lat float64
	time     time.Time
}

func (p position) isEmpty() bool {
	return p == position{}
}

func (p position) String() string {
	return fmt.Sprintf("{lon: %v, lat: %v, t: %s}", p.lon, p.lat, p.time.String())
}

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
	e.GET("/position", getPosition)

	e.Logger.Fatal(e.StartTLS(":8023", "cert.pem", "key.pem"))
}

func health(c echo.Context) error {
	var data bytes.Buffer
	healthTmpl.Execute(&data, c.Request())
	return c.HTML(http.StatusOK, data.String())
}

func getPosition(c echo.Context) error {
	if lastPosition.isEmpty() {
		return c.NoContent(http.StatusServiceUnavailable)
	}

	return c.HTML(http.StatusOK, fmt.Sprintf(`{"lon":"%v","lat":"%v"}`, lastPosition.lon, lastPosition.lat))
}

func postPosition(c echo.Context) error {
	data := new(bytes.Buffer)
	data.ReadFrom(c.Request().Body)

	var points []float64
	pairs := strings.Split(data.String(), "&")
	for _, pair := range pairs {
		if strings.HasPrefix(pair, "points=") {
			values := strings.Split(pair[7:], "+")
			for _, value := range values {
				point, _ := strconv.ParseFloat(value, 64)
				points = append(points, point)
			}
			break
		}
	}

	//	log.Printf("request: %v", data)
	//	log.Printf("points: %v", points)

	lastPosition = position{lat: points[0], lon: points[1], time: time.Now()}
	log.Printf("Got position: %s", lastPosition)

	return c.NoContent(http.StatusOK)
}
