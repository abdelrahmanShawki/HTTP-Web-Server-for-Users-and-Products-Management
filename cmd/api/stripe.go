package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/paymentintent"
	"github.com/stripe/stripe-go/v72/webhook"
	_ "interviewTask/internal/data"
	"io"
	"net/http"
	"os"
	"time"
)

// process payments
func (app *application) processStripePayment(amount float64, userId int64) (string, error) {

	// Convert amount to the smallest currency unit (cents for USD).
	amountCents := int64(amount * 100)

	// Set your Stripe secret key from the configuration.
	stripe.Key = app.config.stripeSecretKey //  .env
	// prevent duplicate purchase
	idempotencyKey := fmt.Sprintf("orderTime__%s__userID__%v", time.Now().Format(time.RFC3339), userId)
	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(amountCents),
		Currency: stripe.String(string(stripe.CurrencyUSD)),
	}
	params.IdempotencyKey = stripe.String(idempotencyKey)

	pi, err := paymentintent.New(params)
	if err != nil {
		return "", fmt.Errorf("stripe payment creation failed: %w", err)
	}

	return pi.ID, nil
}

// listen to stripe
func (app *application) stripeWebhookHandler(w http.ResponseWriter, r *http.Request) {
	// Read the raw body.
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// Retrieve and verify the Stripe signature.
	sigHeader := r.Header.Get("Stripe-Signature")
	endpointSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")
	if endpointSecret == "" {
		app.serverErrorResponse(w, r, errors.New("missing STRIPE_WEBHOOK_SECRET"))
		return
	}

	event, err := webhook.ConstructEvent(payload, sigHeader, endpointSecret)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// Process the event based on its type.
	switch event.Type {
	case "payment_intent.succeeded":
		var pi stripe.PaymentIntent
		if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
			app.badRequestResponse(w, r, err)
			return
		}
		// Update order status to "paid" using the PaymentIntent ID.
		if err := app.models.Orders.UpdateStatusByStripePaymentID(pi.ID, "paid"); err != nil {
			// Log the error or handle it appropriately.
			app.logger.PrintError(err, map[string]string{"payment_intent_id": pi.ID})
		}
		app.logger.PrintInfo("payment_intent succeeded", map[string]string{"payment_intent_id": pi.ID})

	case "payment_intent.payment_failed":
		var pi stripe.PaymentIntent
		if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
			app.badRequestResponse(w, r, err)
			return
		}
		// Update order status to "failed" based on the PaymentIntent ID.
		if err := app.models.Orders.UpdateStatusByStripePaymentID(pi.ID, "failed"); err != nil {
			app.logger.PrintError(err, map[string]string{"payment_intent_id": pi.ID})
		}
		app.logger.PrintInfo("payment_intent failed", map[string]string{"payment_intent_id": pi.ID})
	default:
		app.logger.PrintInfo("unhandled event type", map[string]string{"type": event.Type})
	}

	// Acknowledge receipt of the event.
	w.WriteHeader(http.StatusOK)
}
