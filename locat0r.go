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

	influx "github.com/influxdata/influxdb/client/v2"
	"github.com/labstack/echo"
	"github.com/twpayne/go-gpx"
)

var healthTmpl = template.New("health")

const (
	addr   = "http://influxdb:8086"
	db     = "locat0r"
	user   = "admin"
	passwd = "geheim"
)

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
		Remote address: {{.Remoteaddr}}<br>
		Method: {{.Method}}<br>
		Path: {{.URL.Path}}<br>
	</code>
	`)

	con, err := connectInflux()
	if err != nil {
		log.Fatalf("Failed to connect to %v: %v", addr, err)
	}
	q := influx.Query{Command: "create database " + db, Database: db}
	_, err = con.Query(q)
	if err != nil {
		log.Fatalf("Could not create database %v: %v", db, err)
	}
	con.Close()

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

	waypoints := queryWaypoints(1)
	position := position{Lon: waypoints[0].Lon, Lat: waypoints[0].Lat, Time: waypoints[0].Time}

	jsonPos, err := json.Marshal(position)
	if err != nil {
		return c.HTML(http.StatusInternalServerError, "Failed to marshall position")
	}

	return c.HTMLBlob(http.StatusOK, jsonPos)
}

func getTrack(c echo.Context) error {

	waypoints := queryWaypoints(100)

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
		savePosition(&position)
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

func savePosition(p *position) {
	c, err := connectInflux()
	if err != nil {
		log.Printf("Error connecting influx at %v: %v", addr, err)
		return
	}
	defer c.Close()

	bp, err := influx.NewBatchPoints(influx.BatchPointsConfig{
		Database:  db,
		Precision: "us"})

	tags := map[string]string{"position": "gps-position"}
	fields := map[string]interface{}{
		"lat": p.Lat,
		"lon": p.Lon}

	pt, err := influx.NewPoint("position", tags, fields, p.Time)
	if err != nil {
		log.Printf("Error creating new point: %v", err)
	}
	bp.AddPoint(pt)

	if err = c.Write(bp); err != nil {
		log.Printf("Error writing position: %v", err)
	}
}

func connectInflux() (influx.Client, error) {
	return influx.NewHTTPClient(influx.HTTPConfig{
		Addr:     addr,
		Username: user,
		Password: passwd})
}

func queryWaypoints(n int) []*gpx.WptType {
	c, err := connectInflux()
	if err != nil {
		log.Printf("Failed to connect influx for querying: %v", err)
	}
	defer c.Close()

	var cmd = fmt.Sprintf(`SELECT "lat", "lon" FROM "locat0r"."autogen"."position" order by time desc limit %d`, n)
	q := influx.Query{Command: cmd, Database: db}

	res, err := c.Query(q)
	if err != nil {
		log.Printf("Failed to query %v: %v", cmd, err)
	}

	var waypoints []*gpx.WptType

	for _, r := range res.Results[0].Series[0].Values {
		time, _ := time.Parse(time.RFC3339, r[0].(string))
		lat, _ := r[1].(json.Number).Float64()
		lon, _ := r[2].(json.Number).Float64()
		waypoints = append(waypoints, &gpx.WptType{
			Lat: lat, Lon: lon, Time: time})
		// log.Printf("i: %v, time: %v, lat: %v, lon: %v", i, r[0], r[1], r[2])
	}

	return waypoints
}
