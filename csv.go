package api2

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"io"
	"net/http"
)

type CsvResponse struct {
	HttpCode    int
	HttpHeaders http.Header
	CsvHeader   []string

	// The channel is used to pass rows of CSV file.
	//
	// In ResponseEncoder (server side) the field should be set by the handler
	// and results written from a goroutine. The transport drains the channel.
	//
	// In ResponseDecoder (client side) the channel should be passed in response
	// object. The transport downloads the file and writes rows to the channel
	// and closes it before returning.
	Rows chan []string
}

func csvEncodeResponse(ctx context.Context, w http.ResponseWriter, res0 interface{}) error {
	res := res0.(*CsvResponse)

	defer func() {
		// Drain the channel.
		for range res.Rows {
		}
	}()

	w.Header().Set("Content-Type", "text/csv")

	// Copy HTTP headers.
	for k, v := range res.HttpHeaders {
		w.Header()[k] = v
	}

	w.WriteHeader(res.HttpCode)

	httpFlusher, hasFlusher := w.(http.Flusher)

	csvWriter := csv.NewWriter(w)
	csvWriter.UseCRLF = true

	if err := csvWriter.Write(res.CsvHeader); err != nil {
		return err
	}
	csvWriter.Flush()
	if hasFlusher {
		httpFlusher.Flush()
	}
	if err := csvWriter.Error(); err != nil {
		return err
	}

	for record := range res.Rows {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := csvWriter.Write(record); err != nil {
			return err
		}
		csvWriter.Flush()
		if hasFlusher {
			httpFlusher.Flush()
		}
		if err := csvWriter.Error(); err != nil {
			return err
		}
	}

	return nil
}

func csvDecodeResponse(ctx context.Context, r *http.Response, res0 interface{}) error {
	res := res0.(*CsvResponse)

	if res.Rows == nil {
		panic("provide a channel in res.Rows")
	}

	defer close(res.Rows)

	res.HttpCode = r.StatusCode

	// Copy HTTP headers.
	res.HttpHeaders = make(http.Header)
	for k, v := range r.Header {
		res.HttpHeaders[k] = v
	}

	csvReader := csv.NewReader(r.Body)

	csvHeader, err := csvReader.Read()
	if err != nil {
		return err
	}
	res.CsvHeader = csvHeader

	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		record, err := csvReader.Read()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		res.Rows <- record
	}
}

func csvEncodeError(ctx context.Context, w http.ResponseWriter, err error) error {
	code := http.StatusBadRequest
	var httpErr HttpError
	if errors.As(err, &httpErr) {
		code = httpErr.HttpCode()
	}
	http.Error(w, err.Error(), code)
	return nil
}

func csvDecodeError(ctx context.Context, res *http.Response) error {
	message, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	return httpError{
		Code:    res.StatusCode,
		Message: string(bytes.TrimSpace(message)),
	}
}

var CsvTransport = &JsonTransport{
	ResponseEncoder: csvEncodeResponse,
	ResponseDecoder: csvDecodeResponse,
	ErrorEncoder:    csvEncodeError,
	ErrorDecoder:    csvDecodeError,
}
