/*
Package popbuilder is the server component of the popbuilder application,
a web-app that allows the user to build a population estimate for an
arbitrary set of small areas in Great Britain.
*/
package main

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"github.com/olihawkins/decimals"
	"github.com/olihawkins/handlers"
	htmlTemplate "html/template"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	textTemplate "text/template"
	"time"
)

// Define package constants
const (
	baseUrl string = "/"
	sep     string = string(filepath.Separator)

	dbDir        string = "db"
	templateDir  string = "templates"
	resourcesDir string = "resources"

	resultsDbPath  string = dbDir + sep + "popzones-10.db"
	downloadDbPath string = dbDir + sep + "popzones-5.db"

	introPath    string = templateDir + sep + "intro.html"
	mapPath      string = templateDir + sep + "map.html"
	resultsPath  string = templateDir + sep + "results.html"
	downloadPath string = templateDir + sep + "download.txt"

	notFoundPath        string = templateDir + sep + "notfound.html"
	errorPath           string = templateDir + sep + "error.html"
	defaultErrorMessage string = "Sorry! An error has occurred."
)

// HomeHandler implements http.Handler and serves requests for the homepage.
type HomeHandler struct {
	introPage       []byte
	mapPage         []byte
	seenCookie      string
	skipCookie      string
	postedForm      string
	skipForm        string
	notFoundHandler *handlers.NotFoundHandler
}

// HomeHandler returns a new homeHandler with the handler values initialised.
func NewHomeHandler(introPath string, mapPath string,
	notFoundHandler *handlers.NotFoundHandler) *HomeHandler {

	// Load the intro page
	introPage, err := ioutil.ReadFile(introPath)

	if err != nil {
		log.Fatal(err)
	}

	// Load the map page
	mapPage, err := ioutil.ReadFile(mapPath)

	if err != nil {
		log.Fatal(err)
	}

	return &HomeHandler{

		introPage:       introPage,
		mapPage:         mapPage,
		seenCookie:      "seen",
		skipCookie:      "skip",
		postedForm:      "posted",
		skipForm:        "skipintro",
		notFoundHandler: notFoundHandler,
	}
}

// ServeHTTP determines whether to serve the intro page or the map page.
func (h *HomeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	// Respond to all requests for unkown paths with a 404
	if r.URL.Path != "/" {

		h.notFoundHandler.ServeHTTP(w, r)
		return
	}

	// The file that will be served
	var page []byte

	// Check if the form was posted
	if posted := r.PostFormValue(h.postedForm); posted != "" {

		// Check if the user wants to skip the intro
		if skip := r.PostFormValue(h.skipForm); skip != "" {

			//Set a cookie for one year to skip the intro
			expires := time.Now().Add(time.Duration(31104000) * time.Second)
			cookie := &http.Cookie{Name: h.skipCookie, Expires: expires}
			http.SetCookie(w, cookie)

		} else {

			//Set a cookie for one hour to indicate the intro screen was seen
			expires := time.Now().Add(time.Duration(3600) * time.Second)
			cookie := &http.Cookie{Name: h.seenCookie, Expires: expires}
			http.SetCookie(w, cookie)
		}

		http.Redirect(w, r, baseUrl, http.StatusFound)
		return

	} else {

		// If the form was not posted, check if the intro should be skipped
		if _, err := r.Cookie(h.skipCookie); err == nil {

			page = h.mapPage

		} else {

			// If seenCookie is set, set it to expired and send the map
			if seenCookie, err := r.Cookie(h.seenCookie); err == nil {

				seenCookie.MaxAge = -1
				http.SetCookie(w, seenCookie)
				page = h.mapPage

			} else {

				// Otherwise, send the intro
				page = h.introPage
			}
		}
	}

	// Serve the given page
	fmt.Fprintf(w, "%s", page)
	return
}

// ResultsData holds population data for a set of zones for the results page.
type ResultsData struct {
	Population string
	Zones      string
	M0, M10, M20, M30, M40, M50, M60, M70, M80, M90,
	F0, F10, F20, F30, F40, F50, F60, F70, F80, F90 int64
}

// ResultsDb encapsulates the sqlite database used by resultsHandler.
type ResultsDb struct {
	db        *sql.DB
	baseQuery string
}

// NewResultsDb returns a new resultsDB with the database initialised.
func NewResultsDb(dbPath string) *ResultsDb {

	// Create a database handle
	dbHandle, err := sql.Open("sqlite3", dbPath)

	if err != nil {
		log.Fatal(err)
	}

	// Create a new resultsDB with the database handle and return a pointer
	return &ResultsDb{

		db: dbHandle,
		baseQuery: `
SELECT
	sum(m_0_9), sum(m_10_19), sum(m_20_29), sum(m_30_39), sum(m_40_49), 
	sum(m_50_59), sum(m_60_69), sum(m_70_79),sum(m_80_89), sum(m_90), 
	sum(f_0_9), sum(f_10_19), sum(f_20_29), sum(f_30_39), sum(f_40_49), 
	sum(f_50_59), sum(f_60_69), sum(f_70_79), sum(f_80_89), sum(f_90)
FROM 
	population 
WHERE 
	code IN (`,
	}
}

// Close closes the database handle held by the resultsDB.
func (r *ResultsDb) Close() {

	r.db.Close()
}

// GetPopulationData returns the population data for the given zones.
func (r *ResultsDb) GetPopulationData(zones []string) (*ResultsData, error) {

	// Declare variables to hold query results
	var m0, m10, m20, m30, m40, m50, m60, m70, m80, m90,
		f0, f10, f20, f30, f40, f50, f60, f70, f80, f90 int64

	// Build the query string and an interface slice of args to pass to Query
	query := r.baseQuery
	args := []interface{}{}

	for i := 0; i < len(zones); i++ {

		query += "?,"
		args = append(args, zones[i])
	}

	query = query[:len(query)-1] + ")"

	// Execute the query and scan the results
	err := r.db.QueryRow(query, args...).Scan(
		&m0, &m10, &m20, &m30, &m40, &m50, &m60, &m70, &m80, &m90,
		&f0, &f10, &f20, &f30, &f40, &f50, &f60, &f70, &f80, &f90)

	if err != nil {
		return nil, err
	}

	// Calculate the total population
	population :=
		m0 + m10 + m20 + m30 + m40 + m50 + m60 + m70 + m80 + m90 +
			f0 + f10 + f20 + f30 + f40 + f50 + f60 + f70 + f80 + f90

	results := &ResultsData{
		Population: decimals.FormatThousands(population),
		M0:         m0, M10: m10, M20: m20, M30: m30, M40: m40,
		M50: m50, M60: m60, M70: m70, M80: m80, M90: m90,
		F0: f0, F10: f10, F20: f20, F30: f30, F40: f40,
		F50: f50, F60: f60, F70: f70, F80: f80, F90: f90,
	}

	return results, nil
}

// ResultsHandler handles requests sent to the results page.
type ResultsHandler struct {
	rdb          *ResultsDb
	errorHandler *handlers.ErrorHandler
	template     *htmlTemplate.Template
	zoneForm     string
}

// NewResultsHandler returns a new ResultsHandler with the values initialised.
func NewResultsHandler(templatePath string, database *ResultsDb,
	errorHandler *handlers.ErrorHandler) *ResultsHandler {

	templateFile, err := htmlTemplate.ParseFiles(templatePath)

	if err != nil {
		log.Fatal(err)
	}

	return &ResultsHandler{

		rdb:          database,
		errorHandler: errorHandler,
		template:     templateFile,
		zoneForm:     "zones",
	}
}

// ServeHTTP expects a list of area codes for population zones as POST data.
// The population data for the given areas is retrieved from a sqlite database
// and is inserted into the template for display in a d3 population pyramid.
func (h *ResultsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	if zonestr := r.PostFormValue(h.zoneForm); zonestr != "" {

		zones := strings.Split(zonestr, ",")
		templateData, err := h.rdb.GetPopulationData(zones)

		if err != nil {

			h.errorHandler.ServeError(w,
				"Could not get population data from the ResultsDb.")

			return
		}

		templateData.Zones = zonestr
		h.template.Execute(w, templateData)

	} else {

		http.Redirect(w, r, baseUrl, http.StatusFound)
	}

	return
}

// DownloadData holds population data for each zone for the download page.
type DownloadData struct {
	Code string
	P0, P5, P10, P15, P20, P25, P30, P35, P40, P45,
	P50, P55, P60, P65, P70, P75, P80, P85, P90,
	M0, M5, M10, M15, M20, M25, M30, M35, M40, M45,
	M50, M55, M60, M65, M70, M75, M80, M85, M90,
	F0, F5, F10, F15, F20, F25, F30, F35, F40, F45,
	F50, F55, F60, F65, F70, F75, F80, F85, F90 int64
}

// DownloadDb encapsulates the sqlite database used by DownloadHandler
type DownloadDb struct {
	db        *sql.DB
	baseQuery string
}

// DownloadDb returns a new DownloadDb with the database initialised.
func NewDownloadDb(dbPath string) *DownloadDb {

	// Create a database handle
	dbHandle, err := sql.Open("sqlite3", dbPath)

	if err != nil {
		log.Fatal(err)
	}

	// Create a new DownloadDb with the database handle and return a pointer
	return &DownloadDb{

		db: dbHandle,
		baseQuery: `
SELECT
	code, p_0_4, p_5_9, p_10_14, p_15_19, p_20_24, p_25_29, p_30_34, p_35_39, 
	p_40_44, p_45_49, p_50_54, p_55_59, p_60_64, p_65_69, p_70_74, p_75_79, 
	p_80_84, p_85_89, p_90, m_0_4, m_5_9, m_10_14, m_15_19, m_20_24, m_25_29, 
	m_30_34, m_35_39, m_40_44, m_45_49, m_50_54, m_55_59, m_60_64, m_65_69, 
	m_70_74, m_75_79, m_80_84, m_85_89, m_90, f_0_4, f_5_9, f_10_14, f_15_19, 
	f_20_24, f_25_29, f_30_34, f_35_39, f_40_44, f_45_49, f_50_54, f_55_59, 
	f_60_64, f_65_69, f_70_74, f_75_79, f_80_84, f_85_89, f_90
FROM 
	population 
WHERE 
	code IN (`,
	}
}

// Close closes the database handle held by the DownloadDb.
func (d *DownloadDb) Close() {

	d.db.Close()
}

// GetPopulationData returns the population data for the given zones.
func (d *DownloadDb) GetPopulationData(zones []string) ([]*DownloadData, error) {

	// Declare variables to hold query results
	var code string
	var row *DownloadData
	var p0, p5, p10, p15, p20, p25, p30, p35, p40, p45,
		p50, p55, p60, p65, p70, p75, p80, p85, p90,
		m0, m5, m10, m15, m20, m25, m30, m35, m40, m45,
		m50, m55, m60, m65, m70, m75, m80, m85, m90,
		f0, f5, f10, f15, f20, f25, f30, f35, f40, f45,
		f50, f55, f60, f65, f70, f75, f80, f85, f90 int64

	// Build the query string and an interface slice of args to pass to Query
	query := d.baseQuery
	args := []interface{}{}

	for i := 0; i < len(zones); i++ {

		query += "?,"
		args = append(args, zones[i])
	}

	query = query[:len(query)-1] + ")"

	// Execute the query and scan the results
	rows, err := d.db.Query(query, args...)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	// Create the results map
	results := []*DownloadData{}

	// Scan the results
	for rows.Next() {

		err := rows.Scan(
			&code, &p0, &p5, &p10, &p15, &p20, &p25, &p30, &p35, &p40, &p45,
			&p50, &p55, &p60, &p65, &p70, &p75, &p80, &p85, &p90,
			&m0, &m5, &m10, &m15, &m20, &m25, &m30, &m35, &m40, &m45,
			&m50, &m55, &m60, &m65, &m70, &m75, &m80, &m85, &m90,
			&f0, &f5, &f10, &f15, &f20, &f25, &f30, &f35, &f40, &f45,
			&f50, &f55, &f60, &f65, &f70, &f75, &f80, &f85, &f90)

		if err != nil {
			return nil, err
		}

		row = &DownloadData{
			Code: code,
			P0:   p0, P5: p5, P10: p10, P15: p15, P20: p20,
			P25: p25, P30: p30, P35: p35, P40: p40, P45: p45,
			P50: p50, P55: p55, P60: p60, P65: p65, P70: p70,
			P75: p75, P80: p80, P85: p85, P90: p90,
			M0: m0, M5: m5, M10: m10, M15: m15, M20: m20,
			M25: m25, M30: m30, M35: m35, M40: m40, M45: m45,
			M50: m50, M55: m55, M60: m60, M65: m65, M70: m70,
			M75: m75, M80: m80, M85: m85, M90: m90,
			F0: f0, F5: f5, F10: f10, F15: f15, F20: f20,
			F25: f25, F30: f30, F35: f35, F40: f40, F45: f45,
			F50: f50, F55: f55, F60: f60, F65: f65, F70: f70,
			F75: f75, F80: f80, F85: f85, F90: f90,
		}

		results = append(results, row)
	}

	err = rows.Err()

	if err != nil {
		return nil, err
	}

	return results, nil
}

// DownloadHandler handles requests sent to the results page.
type DownloadHandler struct {
	ddb          *DownloadDb
	errorHandler *handlers.ErrorHandler
	template     *textTemplate.Template
	zoneForm     string
}

// DownloadHandler returns a new homeHandler with the values initialised.
func NewDownloadHandler(templatePath string, database *DownloadDb,
	errorHandler *handlers.ErrorHandler) *DownloadHandler {

	templateFile, err := textTemplate.ParseFiles(templatePath)

	if err != nil {
		log.Fatal(err)
	}

	return &DownloadHandler{

		ddb:          database,
		errorHandler: errorHandler,
		template:     templateFile,
		zoneForm:     "zones",
	}
}

// ServeHTTP expects a list of area codes for population zones as POST data.
// The population data for the given areas is retrieved from a sqlite database
// and is sent to the browser as a csv download.
func (h *DownloadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	if zonestr := r.PostFormValue(h.zoneForm); zonestr != "" {

		zones := strings.Split(zonestr, ",")
		templateData, err := h.ddb.GetPopulationData(zones)

		if err != nil {

			h.errorHandler.ServeError(w,
				"Could not get population data from the DownloadDb.")

			return
		}

		// Set headers to mark it as a file download
		w.Header().Set("Content-Disposition", "attachment; filename=download.csv")
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")

		// These headers are needed for the download to work in older versions
		// of IE. Add a user-agent check if this causes problems in other browsers.
		w.Header().Set("Cache-Control", "must-revalidate, post-check=0, pre-check=0")
		w.Header().Set("Pragma", "public")

		h.template.Execute(w, templateData)

	} else {

		http.Redirect(w, r, baseUrl, http.StatusFound)
	}

	return
}

func main() {

	// Set the port number
	portNumber := 3000
	portString := fmt.Sprint(":", portNumber)

	// Create a ResultsDb for the results page
	resultsDb := NewResultsDb(resultsDbPath)
	defer resultsDb.Close()

	// Create a DownloadDb for the download page
	downloadDb := NewDownloadDb(downloadDbPath)
	defer downloadDb.Close()

	// Create the utility handlers
	notFoundHandler := handlers.LoadNotFoundHandler(notFoundPath)
	errorHandler := handlers.LoadErrorHandler(errorPath, defaultErrorMessage, true)

	// Create the the page handlers for home, results and download pages
	http.Handle("/", NewHomeHandler(introPath, mapPath, notFoundHandler))
	http.Handle("/results", NewResultsHandler(resultsPath, resultsDb, errorHandler))
	http.Handle("/download", NewDownloadHandler(downloadPath, downloadDb, errorHandler))

	// Create a filehandler to a static directory
	fileHandler := handlers.NewFileHandler("/resources/", resourcesDir, notFoundHandler)
	http.Handle("/resources/", fileHandler)

	// Start server
	log.Print("Server starting on port ", portNumber, " ...")
	err := http.ListenAndServe(portString, nil)

	if err != nil {
		log.Fatal(err)
	}

	return
}
