package main

import (
	"errors"
	"interviewTask/internal/data"
	"net/http"
	"time"
)

// - >> just a reminder ->>> when testing the addcreditcard from postman , make sure that token in the header matches
// the token returned from last login to avoid any authentication errors .

func (app *application) AddCreditCard(w http.ResponseWriter, r *http.Request) {
	// Define a struct to capture the expected JSON payload.
	var input struct {
		CardToken      string `json:"card_token"`
		ExpiryDate     string `json:"expiry_date"`     // expected in "2006-01-02" format
		CardholderName string `json:"cardholder_name"` // optional
	}

	// Decode the JSON request.
	err := app.readJson(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// Retrieve the authenticated user ID from the request context.
	userID, ok := r.Context().Value("userID").(int64)
	if !ok {
		app.invalidCredentialsResponse(w, r)
		return
	}
	// Parse the expiry date.
	expiry, err := time.Parse("2006-01-02", input.ExpiryDate)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// Create a new CreditCard instance.
	card := &data.CreditCard{
		UserID:         userID,
		CardToken:      input.CardToken,
		ExpiryDate:     expiry,
		CardholderName: input.CardholderName,
	}

	if err = app.models.Creditcard.Insert(card); err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Respond with the newly created credit card.
	app.writeJson(w, http.StatusCreated, envelope{"credit_card": card}, nil)
}

func (app *application) DeleteCreditCard(w http.ResponseWriter, r *http.Request) {
	// Define a struct to capture the expected JSON payload.
	var input struct {
		ID int64 `json:"id"`
	}

	// Decode the JSON request.
	err := app.readJson(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// Retrieve the authenticated user ID from the request context.

	userID, ok := r.Context().Value("userID").(int64)
	if !ok {
		app.invalidCredentialsResponse(w, r)
		return
	}

	err = app.models.Creditcard.Delete(input.ID, userID)
	if err != nil {
		// Check if the error is due to a missing record.
		if errors.Is(err, data.ErrRecordNotFound) {
			app.notFoundResponse(w, r)
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Respond with a success message.
	app.writeJson(w, http.StatusOK, envelope{"message": "credit card deleted successfully"}, nil)
}
