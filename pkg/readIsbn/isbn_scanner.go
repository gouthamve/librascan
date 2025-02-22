package readIsbn

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gouthamve/librascan/pkg/models"
	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	currentShelfGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "librascan_current_shelf",
		Help: "The current shelf and row",
	}, []string{"shelf", "shelfID", "row"})

	booksProcessedCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "librascan_books_processed",
		Help: "The total number of books processed",
	})

	booksFailedCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "librascan_books_failed",
		Help: "The total number of books that failed to process",
	})

	librascanAPIRequests = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "librascan_api_requests",
		Help:    "Histogram of API requests",
		Buckets: prometheus.DefBuckets,
	}, []string{"code", "method"})
)

func StartCLI(serverURL string, hookDevice bool) {
	// Start an echo server and run Prometheus.
	go func() {
		e := echo.New()
		e.Use(echoprometheus.NewMiddleware("librascan"))
		e.GET("/metrics", echoprometheus.NewHandler())
		e.Logger.Fatal(e.Start(":8080"))
	}()

	client := &http.Client{
		Transport: http.DefaultTransport,
	}
	client.Transport = promhttp.InstrumentRoundTripperDuration(librascanAPIRequests, client.Transport)

	inputLoop(client, serverURL, hookDevice)
}

func inputLoop(httpClient *http.Client, serverURL string, hookDevice bool) {
	shelf, rowNumber, err := getShelfFromCode(httpClient, serverURL, "00000")
	if err != nil {
		log.Fatalln("Cannot get shelf:", err)
	}
	currentShelfGauge.WithLabelValues(shelf.Name, strconv.Itoa(shelf.ID), strconv.Itoa(rowNumber)).Set(1)

	var getInput func() string

	getInput = func() string {
		var input string
		fmt.Scanln(&input)
		return input
	}

	if hookDevice {
		device, err := grabAndSetupDevice()
		if err != nil {
			log.Fatalln("Cannot open device:", err)
		}

		getInput = func() string {
			return device.read()
		}
	}

	for {
		fmt.Println("Enter ISBN13 or shelfCode: ")

		input := getInput()

		fmt.Println("Input:", input)

		// EAN Codes can be 8 or 13 digits long.
		// We are using the 8 digit EAN codes for shelf codes.
		if len(input) == 8 {
			prevShelf := shelf
			prevRow := rowNumber

			shelf, rowNumber, err = getShelfFromCode(httpClient, serverURL, input)
			if err != nil {
				slog.Error("cannot get shelf; using previous shelf", "error", err, "prev_shelf", prevShelf.Name)
				shelf = prevShelf
				rowNumber = prevRow
			} else {
				currentShelfGauge.WithLabelValues(shelf.Name, strconv.Itoa(shelf.ID), strconv.Itoa(rowNumber)).Set(1)
				slog.Info("Shelf changed", "shelf", shelf.Name, "row", rowNumber)
			}

			continue
		}

		if len(input) != 13 {
			fmt.Println("Invalid ISBN")
			continue
		}

		fmt.Println("ISBN:", input, "Shelf:", shelf.Name, "Row:", rowNumber)
		booksProcessedCounter.Inc()
		ingestBook(httpClient, serverURL, input, shelf.ID, rowNumber)
	}
}

func ingestBook(httpClient *http.Client, serverURL, isbn string, shelfID, rowNumber int) {
	// Use the provided serverURL instead of the hardcoded value.
	fullURL := fmt.Sprintf("%s/books/%s?shelf_id=%d&row_number=%d", serverURL, isbn, shelfID, rowNumber)
	resp, err := httpClient.Post(fullURL, "application/json", io.Reader(nil))
	if err != nil {
		slog.Error("cannot post ISBN", "error", err)
		booksFailedCounter.Inc()
		return
	}

	if resp.StatusCode/100 != 2 {
		slog.Error("unexpected status code", "status", resp.StatusCode)
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			slog.Error("cannot read response body", "error", err)
		}
		fmt.Println("Body:", string(body))
		booksFailedCounter.Inc()
		return
	} else {
		book := models.Book{}
		if err := json.NewDecoder(resp.Body).Decode(&book); err != nil {
			slog.Error("cannot decode response body", "error", err)
			booksFailedCounter.Inc()
			return
		}

		fmt.Println("Book:", book)
	}

	if err := resp.Body.Close(); err != nil {
		slog.Error("cannot close response body", "error", err)
	}
}

func getShelfFromCode(httpClient *http.Client, serverURL, shelfCodeStr string) (models.Shelf, int, error) {
	shelfCode, err := strconv.Atoi(shelfCodeStr)
	if err != nil {
		return models.Shelf{}, 0, fmt.Errorf("invalid shelf code: %w", err)
	}
	// The last digit is a checksum.
	shelfCode /= 10

	rowNumber := shelfCode % 10
	shelfID := shelfCode / 10

	fullURL := fmt.Sprintf("%s/shelf/%d", serverURL, shelfID)
	resp, err := httpClient.Get(fullURL)
	if err != nil {
		return models.Shelf{}, 0, fmt.Errorf("cannot get shelf: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		return models.Shelf{}, 0, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	shelf := models.Shelf{}
	json.NewDecoder(resp.Body).Decode(&shelf)
	return shelf, rowNumber, nil
}
