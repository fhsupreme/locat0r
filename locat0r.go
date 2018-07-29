package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo"
	gpx "github.com/twpayne/go-gpx"
)

var healthTmpl = template.New("health")

type position struct {
	Lon  float64   `json:"lon"`
	Lat  float64   `json:"lat"`
	Time time.Time `json:"time"`
}

func (p position) isEmpty() bool {
	return p == position{}
}

func (p position) String() string {
	return fmt.Sprintf("{lon: %v, lat: %v, t: %s}", p.Lon, p.Lat, p.Time.String())
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
	e.GET("/track", getTrack)

	e.Logger.Fatal(e.StartTLS(":8023", "cert.pem", "key.pem"))
}

func health(c echo.Context) error {
	var data bytes.Buffer
	healthTmpl.Execute(&data, c.Request())
	return c.HTML(http.StatusOK, data.String())
}

func getPosition(c echo.Context) error {

	position := position{Lon: 11.2914025, Lat: 48.6810944, Time: time.Now()}

	jsonPos, err := json.Marshal(position)
	if err != nil {
		return c.HTML(http.StatusInternalServerError, "Failed to marshall position")
	}

	return c.HTMLBlob(http.StatusOK, jsonPos)
}

func getTrack(c echo.Context) error {

	var waypoints []*gpx.WptType

	waypoints = append(waypoints, &gpx.WptType{
		Lat: 48.6855344, Lon: 11.2511425,
	})
	waypoints = append(waypoints, &gpx.WptType{
		Lat: 48.6844522, Lon: 11.2312325,
	})
	waypoints = append(waypoints, &gpx.WptType{
		Lat: 48.6833643, Lon: 11.2213225,
	})
	waypoints = append(waypoints, &gpx.WptType{
		Lat: 48.6821744, Lon: 11.2714125,
	})
	waypoints = append(waypoints, &gpx.WptType{
		Lat: 48.6810944, Lon: 11.2914025,
	})

	g := &gpx.GPX{
		Version: "1.0",
		Creator: "Locat0r",
		Trk: []*gpx.TrkType{{
			TrkSeg: []*gpx.TrkSegType{{
				TrkPt: waypoints,
			}},
		}},
	}
	var data bytes.Buffer
	if err := g.Write(&data); err != nil {
		fmt.Printf("err == %v", err)
	}

	return c.HTMLBlob(http.StatusOK, data.Bytes())
}

func postPosition(c echo.Context) error {
	data := new(bytes.Buffer)
	data.ReadFrom(c.Request().Body)

	go processNewPosition(data.String())
	return c.NoContent(http.StatusOK)
}

func processNewPosition(mapMyTrack string) {
	if position, err := parseMapMyTrack(mapMyTrack); err == nil {
		log.Printf("Got position: %s", position)
	} else {
		log.Printf("Failed to parse position: %v", err)
	}
}

func parseMapMyTrack(data string) (position, error) {
	var points []float64
	pairs := strings.Split(data, "&")
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

	if len(points) < 2 {
		return position{}, errors.New("Unable to parse: " + data)
	}

	return position{Lat: points[0], Lon: points[1], Time: time.Now()}, nil
}
