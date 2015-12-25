package main

import (
	"github.com/olihawkins/handlers"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Test HomeHandler with all possible inputs
func TestHomeHandler(t *testing.T) {

	var (
		h               *HomeHandler
		notFoundHandler *handlers.NotFoundHandler
		err             error
		introPage       []byte
		mapPage         []byte
		introString     string
		mapString       string
		bodyString      string
		cookieString    string
		request         *http.Request
		response        *httptest.ResponseRecorder
		expires         time.Time
		cookie          *http.Cookie
		form            url.Values
	)

	// Create a NotFoundHandler for the 404 page
	notFoundHandler = handlers.LoadNotFoundHandler(notFoundPath)

	// Create a HomeHandler to test
	h = NewHomeHandler(introPath, mapPath, notFoundHandler)

	// Load intro page from disk for comparison of output
	introPage, err = ioutil.ReadFile(introPath)

	if err != nil {
		t.Errorf("Could not read intro.html while testing HomeHandler")
	}

	introString = string(introPage)

	// Load map page from disk for comparison of output
	mapPage, err = ioutil.ReadFile(mapPath)

	if err != nil {
		t.Errorf("Could not read map.html while testing HomeHandler")
	}

	mapString = string(mapPage)

	// Test the notFoundHandler returns a 404 for unknown paths
	request, _ = http.NewRequest("GET", "/thispathdoesnotexist", nil)
	response = httptest.NewRecorder()

	h.ServeHTTP(response, request)

	// Check status code
	if response.Code != http.StatusNotFound {
		t.Errorf("Expected StatusNotFound from HomeHandler. Got: %s", response.Code)
	}

	// Test as a new visitor with no cookies set
	request, _ = http.NewRequest("GET", "/", nil)
	response = httptest.NewRecorder()

	h.ServeHTTP(response, request)

	// Check status code
	if response.Code != http.StatusOK {
		t.Errorf("Expected StatusOK from HomeHandler. Got: %s", response.Code)
	}

	// Check handler serves intro.html by comparing with file on disk
	bodyString = response.Body.String()

	if introString != bodyString {
		t.Errorf("Expected intro.html from HomeHandler. Got: %s", bodyString)
	}

	// Test as a visitor who has seen the intro once but not opted out
	request, _ = http.NewRequest("GET", "/", nil)
	response = httptest.NewRecorder()

	expires = time.Now().Add(time.Duration(3600) * time.Second)
	cookie = &http.Cookie{Name: h.seenCookie, Expires: expires}
	request.AddCookie(cookie)

	h.ServeHTTP(response, request)

	// Check status code
	if response.Code != http.StatusOK {
		t.Errorf("Expected StatusOK from HomeHandler. Got: %s", response.Code)
	}

	// Check handler serves map.html by comparing with file on disk
	bodyString = response.Body.String()

	if mapString != bodyString {
		t.Errorf("Expected map.html from HomeHandler. Got: %s", bodyString)
	}

	// Test as a visitor who has seen the intro and opted out
	request, _ = http.NewRequest("GET", "/", nil)
	response = httptest.NewRecorder()

	expires = time.Now().Add(time.Duration(31104000) * time.Second)
	cookie = &http.Cookie{Name: h.skipCookie, Expires: expires}
	request.AddCookie(cookie)

	h.ServeHTTP(response, request)

	// Check status code
	if response.Code != http.StatusOK {
		t.Errorf("Expected StatusOK from HomeHandler. Got: %s", response.Code)
	}

	// Check handler serves map.html by comparing with file on disk
	bodyString = response.Body.String()

	if mapString != bodyString {
		t.Errorf("Expected map.html from HomeHandler. Got: %s", bodyString)
	}

	// Test as a visitor who has clicked continue and not opted out
	form = url.Values{}
	form.Add(h.postedForm, "true")

	request, _ = http.NewRequest("POST", "/", strings.NewReader(form.Encode()))
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Add("Content-Length", strconv.Itoa(len(form.Encode())))
	response = httptest.NewRecorder()

	h.ServeHTTP(response, request)

	// Check status code
	if response.Code != http.StatusFound {
		t.Errorf("Expected StatusFound from HomeHandler. Got: %s", response.Code)
	}

	// Check if the response contains the seenCookie
	cookieString = response.Header()["Set-Cookie"][0]

	if !strings.Contains(cookieString, h.seenCookie) {
		t.Errorf("Expected the seen cookie in header from HomeHandler. "+
			"Got: %s", response.Header())
	}

	// Check if the response contains the redirect to the base url
	if response.Header()["Location"][0] != baseUrl {
		t.Errorf("Expected redirect in header from HomeHandler. Got: %s",
			response.Header())
	}

	// Test as a visitor who has clicked continue and opted out
	form = url.Values{}
	form.Add(h.postedForm, "true")
	form.Add(h.skipForm, "on")

	request, _ = http.NewRequest("POST", "/", strings.NewReader(form.Encode()))
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Add("Content-Length", strconv.Itoa(len(form.Encode())))
	response = httptest.NewRecorder()

	h.ServeHTTP(response, request)

	// Check status code
	if response.Code != http.StatusFound {
		t.Errorf("Expected StatusFound from HomeHandler. Got: %s", response.Code)
	}

	// Check if the response contains the skipCookie
	cookieString = response.Header()["Set-Cookie"][0]

	if !strings.Contains(cookieString, h.skipCookie) {
		t.Errorf("Expected the seen cookie in header from HomeHandler. "+
			"Got: %s", response.Header())
	}

	// Check if the response contains the redirect to the base url
	if response.Header()["Location"][0] != baseUrl {
		t.Errorf("Expected redirect in header from HomeHandler. Got: %s",
			response.Header())
	}
}

// Test ResultsDb.GetPopulationData with a range of inputs.
// Note that these tests check that the method returns correct data.
// The expected results will therefore change whenever the population
// databases change, and this unit test should be updated at that point.
// This test implicitly tests NewResultsDb().
func TestResultsDbGetPopulationData(t *testing.T) {

	codes := [][]string{
		// Test each of these zones in separate queries
		{"E01004736"},
		{"E01004731"},
		{"E01004732"},
		{"E01004733"},
		{"E01004744"},
		{"E01004748"},
		{"E01004743"},
		{"E01004745"},
		{"E01004746"},
		{"E01004747"},
		// Then test them all in one query
		{
			"E01004736",
			"E01004731",
			"E01004732",
			"E01004733",
			"E01004744",
			"E01004748",
			"E01004743",
			"E01004745",
			"E01004746",
			"E01004747",
		},
	}

	expected := []string{
		"1,863",
		"1,927",
		"1,608",
		"2,456",
		"1,375",
		"1,690",
		"1,534",
		"2,341",
		"1,531",
		"2,430",
		"18,755",
	}

	// Create a ResultsDb
	rdb := NewResultsDb(resultsDbPath)
	defer rdb.Close()

	// Test the function output against the expected output
	for i, c := range codes {

		// Get the total population string as returned from the function
		results, err := rdb.GetPopulationData(c)

		if err != nil {
			t.Errorf("Could not get data from ResultsDb.")
		}

		output := results.Population

		if output != expected[i] {

			t.Errorf("Expected %s in ResultsDb.GetPopulationData. Got: %s",
				expected[i], output)

		}
	}
}

// Test ResultsHandler with a range of inputs.
// Note that these tests check that the method returns correct data.
// The expected results will therefore change whenever the population
// databases change, and this unit test should be updated at that point.
// This test implicitly tests NewResultsHandler().
func TestResultsHandler(t *testing.T) {

	var (
		h            *ResultsHandler
		resultsDb    *ResultsDb
		errorHandler *handlers.ErrorHandler
	)

	// Create a ResultsDb
	resultsDb = NewResultsDb(resultsDbPath)
	defer resultsDb.Close()

	// Create an ErrorHandler
	errorHandler = handlers.LoadErrorHandler(errorPath, "", true)

	// Create a ResultsHandler to test
	h = NewResultsHandler(resultsPath, resultsDb, errorHandler)

	codes := []string{
		// Test each of these zones in separate page requests
		"E01004736",
		"E01004731",
		"E01004732",
		"E01004733",
		"E01004744",
		"E01004748",
		"E01004743",
		"E01004745",
		"E01004746",
		"E01004747",
		// Then test them all in one page request
		"E01004736,E01004731,E01004732,E01004733,E01004744," +
			"E01004748,E01004743,E01004745,E01004746,E01004747",
	}

	expected := []string{
		"1,863",
		"1,927",
		"1,608",
		"2,456",
		"1,375",
		"1,690",
		"1,534",
		"2,341",
		"1,531",
		"2,430",
		"18,755",
	}

	// Test the handler output against each expected output
	for i, c := range codes {

		// Submit each set of zone codes as a POST request and check output
		form := url.Values{}
		form.Add(h.zoneForm, c)

		request, _ := http.NewRequest("POST", "/", strings.NewReader(form.Encode()))
		request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		request.Header.Add("Content-Length", strconv.Itoa(len(form.Encode())))
		response := httptest.NewRecorder()

		h.ServeHTTP(response, request)

		// Check status code
		if response.Code != http.StatusOK {
			t.Errorf("Expected StatusOK from resultsHandler. Got: %s",
				response.Code)
		}

		// Check if the response body contains the expected string
		bodyString := response.Body.String()
		expectedString := "The selected population is <b>" + expected[i] + "</b>"

		if !strings.Contains(bodyString, expectedString) {
			t.Errorf("Expected the total in body from resultsHandler. "+
				"Got: %s", bodyString)
		}
	}
}

// Test DownloadDb.GetPopulationData with a range of inputs.
// Note that these tests check that the method returns correct data.
// The expected results will therefore change whenever the population
// databases change, and this unit test should be updated at that point.
// This test implicitly tests NewDownloadDb().
func TestDownloadDbGetPopulationData(t *testing.T) {

	// A checksum of the population data
	var output int64

	codes := [][]string{
		// Test each of these zones in separate queries
		{"E01004736"},
		{"E01004731"},
		{"E01004732"},
		{"E01004733"},
		{"E01004744"},
		{"E01004748"},
		{"E01004743"},
		{"E01004745"},
		{"E01004746"},
		{"E01004747"},
		// Then test them all in one query
		{
			"E01004736",
			"E01004731",
			"E01004732",
			"E01004733",
			"E01004744",
			"E01004748",
			"E01004743",
			"E01004745",
			"E01004746",
			"E01004747",
		},
	}

	expected := []int64{
		49,
		87,
		78,
		84,
		99,
		76,
		103,
		132,
		84,
		117,
		909,
	}

	// Create a DownloadDb
	ddb := NewDownloadDb(downloadDbPath)
	defer ddb.Close()

	// Test the function output against the expected output
	for i, c := range codes {

		populationData, err := ddb.GetPopulationData(c)

		if err != nil {
			t.Errorf("Could not get data from DownloadDb.")
		}

		output = 0

		// Loop through each zone in the dataset and get the first data point
		for _, zoneData := range populationData {

			// Add to the total
			output += zoneData.P0
		}

		if output != expected[i] {

			t.Errorf("Expected %s in DownloadDb.GetPopulationData. Got: %s",
				expected[i], output)

		}
	}
}

// Test DownloadHandler with a range of inputs.
// Note that these tests check that the method returns correct data.
// The expected results will therefore change whenever the population
// databases change, and this unit test should be updated at that point.
// This test implicitly tests NewResultsHandler().
func TestDownloadHandler(t *testing.T) {

	var (
		h            *DownloadHandler
		downloadDb   *DownloadDb
		errorHandler *handlers.ErrorHandler
	)

	// Create a DownloadDb
	downloadDb = NewDownloadDb(downloadDbPath)
	defer downloadDb.Close()

	// Create an ErrorHandler
	errorHandler = handlers.LoadErrorHandler(errorPath, "", true)

	// Create a DownloadHandler to test
	h = NewDownloadHandler(downloadPath, downloadDb, errorHandler)

	codes := []string{
		// Test each of these zones in separate page requests
		"E01004736",
		"E01004731",
		"E01004732",
		"E01004733",
		"E01004744",
		"E01004748",
		"E01004743",
		"E01004745",
		"E01004746",
		"E01004747",
		// Then test them all in one page request
		"E01004736,E01004731,E01004732,E01004733,E01004744," +
			"E01004748,E01004743,E01004745,E01004746,E01004747",
	}

	expected := []string{
		"E01004736,49",
		"E01004731,87",
		"E01004732,78",
		"E01004733,84",
		"E01004744,99",
		"E01004748,76",
		"E01004743,103",
		"E01004745,132",
		"E01004746,84",
		"E01004747,117",
		"E01004736,49",
	}

	// Test the handler output against each expected output
	for i, c := range codes {

		// Submit each set of zone codes as a POST request and check output
		form := url.Values{}
		form.Add(h.zoneForm, c)

		request, _ := http.NewRequest("POST", "/", strings.NewReader(form.Encode()))
		request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		request.Header.Add("Content-Length", strconv.Itoa(len(form.Encode())))
		response := httptest.NewRecorder()

		h.ServeHTTP(response, request)

		// Check status code
		if response.Code != http.StatusOK {
			t.Errorf("Expected StatusOK from downloadHandler. Got: %s",
				response.Code)
		}

		// Check if the response body contains the expected string
		bodyString := response.Body.String()

		if !strings.Contains(bodyString, expected[i]) {
			t.Errorf("Expected the total in body from downloadHandler. "+
				"Got: %s", bodyString)
		}
	}
}
