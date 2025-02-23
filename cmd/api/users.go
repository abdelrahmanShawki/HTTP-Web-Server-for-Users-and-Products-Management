package main

import (
	"errors"
	"fmt"
	"interviewTask/internal/authentication"
	"interviewTask/internal/data"
	"interviewTask/internal/validator"
	"net/http"
)

func (app *application) SignUpUser(w http.ResponseWriter, r *http.Request) {
	// Define an anonymous struct to hold the expected input.
	var input struct {
		FirstName string `json:"first_name"`
		Email     string `json:"email"`
		Role      string `json:"role"`
		Password  string `json:"password"`
	}

	// Read and decode the JSON request body.
	err := app.readJson(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// Sanitize inputs.
	input.FirstName = validator.SanitizeString(input.FirstName)
	input.Email = validator.SanitizeString(input.Email)
	input.Role = validator.SanitizeString(input.Role)

	// Create a new user instance from the input.
	user := &data.User{
		FirstName: input.FirstName,
		Email:     input.Email,
		Role:      input.Role,
	}

	// Set the password (hashing it in the process).
	err = user.Password.Set(input.Password)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Validate the user data.
	v := validator.New()
	data.ValidateUser(v, user)
	if !v.Valid() {
		app.validationErrorResponse(w, r, v.Errors)
		return
	}

	// Insert the new user into the database.
	// (Assumes app.models.Users.Insert(user) is implemented in your data layer.)
	err = app.models.Users.Insert(user)
	if err != nil {
		if errors.Is(err, data.ErrDuplicateEmail) {
			app.errorResponse(w, r, http.StatusConflict, "email already in use")
		} else {
			app.logger.PrintError(err, nil)
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Return the created user as a JSON response.
	// The password field is omitted due to the json:"-" tag.
	app.writeJson(w, http.StatusCreated, envelope{"user": user}, nil)
}

func (app *application) LoginUser(w http.ResponseWriter, r *http.Request) {
	// Define a struct to hold the expected input.
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	// Read and decode the JSON request body.
	err := app.readJson(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// Sanitize input.
	input.Email = validator.SanitizeString(input.Email)

	// Retrieve the user record by email.
	user, err := app.models.Users.GetByEmail(input.Email)
	if err != nil {
		if errors.Is(err, data.ErrRecordNotFound) {
			app.invalidCredentialsResponse(w, r)
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Verify the provided password against the stored hash.
	valid, err := user.Password.Matches(input.Password)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	if !valid {
		app.invalidCredentialsResponse(w, r)
		return
	}

	token, err := auth.GenerateToken(user.ID, user.Role)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	app.writeJson(w, http.StatusOK, envelope{"user": user, "token": token}, nil)
}

func (app *application) GetPurchaseHistory(w http.ResponseWriter, r *http.Request) {
	// Retrieve the authenticated user ID from the request context.
	userID, ok := r.Context().Value(userContextKey).(int64)
	if !ok {
		app.invalidCredentialsResponse(w, r)
		return
	}

	// Retrieve the detailed purchase history (including product info) from the Orders model.
	history, err := app.models.Orders.GetPurchaseHistory(userID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Return the purchase history as a JSON response.
	app.writeJson(w, http.StatusOK, envelope{"purchase_history": history}, nil)
}

func (app application) BuyProducts(w http.ResponseWriter, r *http.Request) {

	// Define the expected JSON payload.
	var input struct {
		Products []struct {
			ID       int64 `json:"id"`
			Quantity int   `json:"quantity"`
		} `json:"products"`
	}

	// Decode the JSON request body.
	err := app.readJson(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// initialize v a new validator , check the len of the products
	v := validator.New()
	v.Check(len(input.Products) > 0, "products", "should be at least one product")
	if !v.Valid() {
		app.validationErrorResponse(w, r, v.Errors)
		return
	}

	// Retrieve the authenticated user ID.
	userID, ok := r.Context().Value(userContextKey).(int64)
	if !ok {
		app.logger.PrintInfo(fmt.Sprintf("user is now is : %s", userID), nil)
		app.invalidCredentialsResponse(w, r)
		return
	}

	// Prepare the order and calculate the total amount.
	order := &data.Order{
		UserID: userID,
	}
	var orderProducts []data.OrderProduct
	var totalAmount float64

	for _, p := range input.Products {
		product, err := app.models.Product.GetByID(p.ID)
		if err != nil {
			app.errorResponse(w, r, http.StatusInternalServerError, "there is no product yet in the db")
			return
		}

		lineTotal := product.Price * float64(p.Quantity)
		totalAmount += lineTotal

		orderProducts = append(orderProducts, data.OrderProduct{
			ProductID:       p.ID,
			Quantity:        p.Quantity,
			PriceAtPurchase: product.Price,
		})
	}

	// Process the payment with Stripe.
	stripePaymentID, err := app.processStripePayment(totalAmount, userID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Set order details.
	order.TotalAmount = totalAmount
	order.StripePaymentID = stripePaymentID

	// Insert the order and associated order_products records.
	err = app.models.Orders.Create(order, orderProducts)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Respond with the order details.
	app.writeJson(w, http.StatusCreated, envelope{
		"order":          order,
		"order_products": orderProducts,
	}, nil)
}
