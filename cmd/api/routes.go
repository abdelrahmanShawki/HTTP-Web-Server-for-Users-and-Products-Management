package main

import (
	"github.com/julienschmidt/httprouter"
	"github.com/justinas/alice"
	_ "interviewTask/internal/authentication"
	"net/http"
)

func (app *application) routes() http.Handler {
	router := httprouter.New()

	// Override default handlers.
	router.NotFound = http.HandlerFunc(app.notFoundResponse)
	router.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowed)

	// Create two chains:
	// All routes need authentication.
	authChain := alice.New(app.AuthMiddleware)
	// Admin routes need both authentication and an admin role check.
	adminChain := alice.New(app.AuthMiddleware, app.RequireRole("admin"))

	//  public routes
	router.HandlerFunc(http.MethodPost, "/user/signup", app.SignUpUser)
	router.HandlerFunc(http.MethodPost, "/user/login", app.LoginUser)
	router.HandlerFunc(http.MethodGet, "/user/products", app.ListProducts) // not tested yet

	//  stripe callback
	router.HandlerFunc(http.MethodPost, "/stripe/webhook", app.stripeWebhookHandler)

	// require authentication.
	router.Handler(http.MethodPost, "/user/credit-card", authChain.Then(http.HandlerFunc(app.AddCreditCard)))
	router.Handler(http.MethodDelete, "/user/credit-card", authChain.Then(http.HandlerFunc(app.DeleteCreditCard)))
	router.Handler(http.MethodPost, "/user/buy", authChain.Then(http.HandlerFunc(app.BuyProducts))) // add some prod to buy
	router.Handler(http.MethodGet, "/user/purchase-history", authChain.Then(http.HandlerFunc(app.GetPurchaseHistory)))

	// Admin endpoints: Require admin privileges.
	router.Handler(http.MethodPost, "/admin/products", adminChain.Then(http.HandlerFunc(app.CreateProduct)))
	router.Handler(http.MethodPut, "/admin/products/:id", adminChain.Then(http.HandlerFunc(app.UpdateProduct)))
	router.Handler(http.MethodDelete, "/admin/products/:id", adminChain.Then(http.HandlerFunc(app.DeleteProduct)))
	router.Handler(http.MethodGet, "/admin/sales", adminChain.Then(http.HandlerFunc(app.SalesFiltering)))

	return router
}
