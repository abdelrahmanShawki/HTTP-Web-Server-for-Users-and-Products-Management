package main

import (
	"errors"
	"fmt"
	"interviewTask/internal/data"
	"interviewTask/internal/validator"
	"net/http"
	"time"
)

func (app application) ListProducts(w http.ResponseWriter, r *http.Request) {

	products, err := app.models.Product.GetAll()
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	app.writeJson(w, http.StatusOK, envelope{"products": products}, nil)
}

// admin handleres  ,, neet to refine some error handling later .
func (app application) CreateProduct(w http.ResponseWriter, r *http.Request) {
	// Retrieve the authenticated userId from the request context.
	userId, ok := r.Context().Value(userContextKey).(int64)
	if !ok {
		app.logger.PrintInfo(fmt.Sprintf("userId is : %s ", userId), nil)
		app.invalidCredentialsResponse(w, r)
		return
	}

	user, err := app.models.Users.GetByID(userId)
	if err != nil {
		app.logger.PrintInfo(fmt.Sprintf("user in create product %s ", user), nil)
		app.serverErrorResponse(w, r, err)
	}
	// Check if the authenticated userId has admin privileges.
	if user.Role != "admin" {
		app.accessDeniedResonse(w, r)
		return
	}

	// Define a struct to capture the expected json .
	var input struct {
		Name           string  `json:"name"`
		Description    string  `json:"description"`
		Price          float64 `json:"price"`
		InventoryCount int     `json:"inventory_count"`
	}

	// Read and decode the json request body.
	err = app.readJson(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// Create a new product instance.
	product := &data.Product{
		Name:           input.Name,
		Description:    input.Description,
		Price:          input.Price,
		InventoryCount: input.InventoryCount,
	}

	// Validate the product.
	v := validator.New()
	data.ValidateProduct(v, product)
	if !v.Valid() {
		app.validationErrorResponse(w, r, v.Errors)
		return
	}

	// Insert the product into the database.
	err = app.models.Product.Create(product)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Respond with the created product.
	app.writeJson(w, http.StatusCreated, envelope{"product": product}, nil)
}

func (app application) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	// Retrieve the product ID from the URL.
	id, err := app.readIDparam(r)
	if err != nil {
		app.errorResponse(w, r, http.StatusBadRequest, "invalid product id")
		return
	}

	// Retrieve the authenticated user ID from the request context.
	userID, ok := r.Context().Value(userContextKey).(int64)
	if !ok {
		app.invalidCredentialsResponse(w, r)
		return
	}

	// Fetch the user from the database to check their role.
	user, err := app.models.Users.GetByID(userID)
	if err != nil {
		if errors.Is(err, data.ErrRecordNotFound) {
			app.invalidCredentialsResponse(w, r)
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Check if the user is an admin.
	if user.Role != "admin" {
		app.accessDeniedResonse(w, r)
		return
	}

	// Define a struct to capture the expected JSON input for updates.
	var input struct {
		Name           *string  `json:"name"`
		Description    *string  `json:"description"`
		Price          *float64 `json:"price"`
		InventoryCount *int     `json:"inventory_count"`
	}

	// Read and decode the JSON request body.
	err = app.readJson(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// Retrieve the current product from the database.
	product, err := app.models.Product.GetByID(id)
	if err != nil {
		if errors.Is(err, data.ErrRecordNotFound) {
			app.notFoundResponse(w, r)
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// assign new product values
	if input.Name != nil {
		product.Name = *input.Name
	}
	if input.Description != nil {
		product.Description = *input.Description
	}
	if input.Price != nil {
		product.Price = *input.Price
	}
	if input.InventoryCount != nil {
		product.InventoryCount = *input.InventoryCount
	}

	// Validate and update the product fields if provided.
	v := validator.New()
	data.ValidateProduct(v, product)
	if !v.Valid() {
		app.validationErrorResponse(w, r, v.Errors)
		return
	}

	// Update the product in the database.
	err = app.models.Product.Update(product)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Respond with the updated product.
	app.writeJson(w, http.StatusOK, envelope{"product": product}, nil)
}

func (app application) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	// Retrieve the product ID from the URL.
	id, err := app.readIDparam(r)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// Retrieve the authenticated user ID from the request context.
	userID, ok := r.Context().Value(userContextKey).(int64)
	if !ok {
		app.invalidCredentialsResponse(w, r)
		return
	}

	// Fetch the user from the database to check their role.
	user, err := app.models.Users.GetByID(userID)
	if err != nil {
		if errors.Is(err, data.ErrRecordNotFound) {
			app.invalidCredentialsResponse(w, r)
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Check if the user is an admin.
	if user.Role != "admin" {
		app.accessDeniedResonse(w, r)
		return
	}

	// Call the data layer to delete the product.
	err = app.models.Product.Delete(id)
	if err != nil {
		// If the product wasn't found, return a not found response.
		if errors.Is(err, data.ErrRecordNotFound) {
			app.notFoundResponse(w, r)
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Respond with a success message.
	app.writeJson(w, http.StatusOK, envelope{"message": "product deleted successfully"}, nil)
}

func (app application) SalesFiltering(w http.ResponseWriter, r *http.Request) {
	// Retrieve the authenticated user ID from the request context.
	userID, ok := r.Context().Value(userContextKey).(int64)
	if !ok {
		app.invalidCredentialsResponse(w, r)
		return
	}

	// Fetch the user from the database to check their role.
	user, err := app.models.Users.GetByID(userID)
	if err != nil {
		if errors.Is(err, data.ErrRecordNotFound) {
			app.invalidCredentialsResponse(w, r)
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Check if the user is an admin.
	if user.Role != "admin" {
		app.invalidCredentialsResponse(w, r)
		return
	}

	// Read query parameters.
	// Expecting "from" and "to" dates in "2006-01-02" format.
	q := r.URL.Query()
	fromStr := q.Get("from")
	toStr := q.Get("to")
	username := q.Get("username") // optional filter

	if fromStr == "" || toStr == "" {
		app.badRequestResponse(w, r, err)
		return
	}

	fromTime, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	toTime, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		app.errorResponse(w, r, http.StatusBadRequest, "invalid 'to' date format; expected YYYY-MM-DD")
		return
	}

	// Call the SalesFiltering method in the Product model.
	sales, err := app.models.Product.SalesFiltering(fromTime, toTime, username)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Return the sales data as JSON.
	app.writeJson(w, http.StatusOK, envelope{"sales": sales}, nil)
}
