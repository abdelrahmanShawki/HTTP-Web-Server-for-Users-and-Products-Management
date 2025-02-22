package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"io"
	"net/http"
	"strconv"
	"strings"
)

func (app *application) readIDparam(r *http.Request) (int64, error) {
	params := httprouter.ParamsFromContext(r.Context())
	id, err := strconv.ParseInt(params.ByName("id"), 10, 64) // base , 64 bit size
	if err != nil || id < 1 {

		return 0, errors.New("invalid id parameter")
	}
	return id, nil
}

type envelope map[string]interface{}

func (app *application) writeJson(w http.ResponseWriter, status int, data envelope, headers http.Header) error {

	//  i want to hilight that json encoder dont use heap memory allocation as same as Marshal
	js, err := json.Marshal(data)
	if err != nil {
		return err
	}
	for key, value := range headers {
		w.Header()[key] = value
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(js)

	return nil
}

func (app *application) readJson(w http.ResponseWriter, r *http.Request, dst interface{}) error {

	// set request max size
	max_bytes := 1_048_500
	r.Body = http.MaxBytesReader(w, r.Body, int64(max_bytes))
	// prepare decoder
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	err := dec.Decode(dst)
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var InvalidUnmarshalTypeError *json.InvalidUnmarshalError

		switch {

		case errors.As(err, &syntaxError):
			return fmt.Errorf("body contains badly-formed JSON (at character %d)", syntaxError.Offset)

		case errors.Is(err, io.ErrUnexpectedEOF):
			return errors.New("body contains badly-formed JSON")

			//data dosnt match the go type
		case errors.As(err, &unmarshalTypeError):
			if unmarshalTypeError.Field != "" {
				return fmt.Errorf("body contains incorrect JSON type for field %q", unmarshalTypeError.Field)
			}
			return fmt.Errorf("body contains incorrect JSON type (at character %d)", unmarshalTypeError.Offset)

		case errors.Is(err, io.EOF):
			return errors.New("body must not be empty")
			// check the error that returned by disableUnknownFields
		case strings.HasPrefix(err.Error(), "json: unknown field"):
			fieldname := strings.TrimPrefix(err.Error(), "json: unknown field")
			_ = fmt.Errorf("this JSON field isnt compatable with our system %s ", fieldname)
			//passing nil or non pointer
		case errors.As(err, &InvalidUnmarshalTypeError):
			panic(err)

			// error when exceeding the header predefined size
		case err.Error() == "http: request body too large":
			return fmt.Errorf("body must not be larger than %d bytes", max_bytes)
		default:
			return err
		}

	}

	// check there is only one struct per request
	err = dec.Decode(&struct{}{})
	if err != io.EOF {
		return errors.New("one JSON Object allowed ")
	}
	return nil
}
