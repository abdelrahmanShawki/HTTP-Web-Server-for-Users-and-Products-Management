# Rescounts Backend Task

This project is an **HTTP web server** for managing **users and products**, developed in **Go**. The server provides a set of RESTful APIs for **user authentication, product management, payment processing via Stripe, and order history retrieval**.

It follows best practices in **security, error handling, and deployment**.

## Features
- User authentication (Sign-up, Login)
- Credit card management (Add, Delete)
- Product catalog (List, Create, Update, Delete)
- Order processing (Purchase products, View purchase history)
- Admin functionalities (Product management, Sales filtering)
- Payment integration with **Stripe**
- Secure **JWT authentication**
- **Fully Dockerized** setup, including database migrations

---

## 📌 Table of Contents
- [Requirements](#requirements)
- [Setup & Installation](#setup--installation)
- [Running the Application](#running-the-application)
- [API Endpoints](#api-endpoints)
- [Environment Variables](#environment-variables)
- [Testing the API (Postman)](#testing-the-api-postman)
- [Project Structure](#project-structure)

---

## 📦 Requirements

Before running this project, ensure you have:

- **Docker & Docker Compose** → [Install Docker](https://docs.docker.com/get-docker/)
- **Stripe account** (for payment integration) → [Create Account](https://stripe.com)

---

## ⚙️ Setup & Installation

Clone the repository and navigate into the project folder:

```sh
git clone https://github.com/abdelrahmanShawki/HTTP-Web-Server-for-Users-and-Products-Management
```

### 1️⃣ Setup Environment Variables

Create a **`.env`** file in the root directory and add:

```env
DSN_BUY_DB=postgres://username:password@db:5432/buy_db?sslmode=disable
JWT_SECRET=your_jwt_secret
STRIPE_SECRET_KEY=your_stripe_secret
STRIPE_WEBHOOK_SECRET=your_stripe_webhook_secret
PORT=4000
POSTGRES_USER=postgres
POSTGRES_PASSWORD=your_password
POSTGRES_DB=buy_db
```

### 2️⃣ Build & Start Services with Docker

Run the following command to build and start the project:

```sh
docker compose up --build
```

> 🚀 This will start **the Go server and PostgreSQL database** in containers and automatically run database migrations.

If you want to **run it in the background**, use:

```sh
docker compose up -d
```

To **stop the application**, run:

```sh
docker compose down
```

---

## ▶️ Running the Application

The application **runs fully on Docker**, including database migrations, so there is **no need for manual migration commands**.

The API will be available at **`http://localhost:4000`**.

If you need to check logs, run:

```sh
docker logs -f app
```

---

## 📡 API Endpoints

| **Endpoint**                     | **Method** | **Description** |
|----------------------------------|------------|----------------|
| `/user/signup`                  | `POST`    | Register a new user |
| `/user/login`                   | `POST`    | Login user & get JWT token |
| `/user/products`                | `GET`     | List available products |
| `/user/buy`                     | `POST`    | Purchase products |
| `/user/purchase-history`        | `GET`     | Get user order history |
| `/user/credit-card`             | `POST`    | Add credit card |
| `/user/credit-card`             | `DELETE`  | Remove credit card |
| `/admin/products`               | `POST`    | Create a product (Admin) |
| `/admin/products/:id`           | `PUT`     | Update a product (Admin) |
| `/admin/products/:id`           | `DELETE`  | Delete a product (Admin) |
| `/admin/sales`                  | `GET`     | Get sales data (Admin) |
| `/stripe/webhook`               | `POST`    | Stripe webhook listener |

> **Authentication:**
> - Most user endpoints require a **Bearer Token** from login.
> - Admin endpoints require a user with the **admin role**.

---

## 🔧 Environment Variables

| Variable                | Description |
|-------------------------|-------------|
| `DSN_BUY_DB`           | PostgreSQL connection string |
| `JWT_SECRET`           | Secret key for JWT authentication |
| `STRIPE_SECRET_KEY`    | Stripe API key for processing payments |
| `STRIPE_WEBHOOK_SECRET`| Stripe webhook secret |
| `PORT`                 | API server port |
| `POSTGRES_USER`        | PostgreSQL username |
| `POSTGRES_PASSWORD`    | PostgreSQL password |
| `POSTGRES_DB`          | PostgreSQL database name |





## 🛠️ Testing the API (Postman)

A **Postman collection** is included in the repository:

1. Open **Postman**.
2. Import the **`rescounts.postman_collection.json`** file.
3. Set up an **environment** with:
   - `{{base_url}} = http://localhost:4000`
   - `{{jwt_token}} = (your JWT token from login)`

---

## 📁 Project Structure

```
📂 rescounts-backend
 ├── 📂 cmd/api          # Main API application
 ├── 📂 internal
 │   ├── 📂 data         # Models & Database interactions
 │   ├── 📂 validator    # Input validation
 │   ├── 📂 authentication # JWT Authentication
 │   ├── 📂 jsonlog      # JSON logging functionality
 ├── 📂 migrations       # Database migration scripts
 ├── 📄 Dockerfile       # Docker configuration
 ├── 📄 docker-compose.yml  # Docker Compose setup
 ├── 📄 entrypoint.sh    # Startup script for Docker
 ├── 📄 .env.example     # Environment variables (example)
 ├── 📄 README.md        # Documentation (this file)
```

---

## 🛡️ Security Considerations

- **JWT-based authentication** to protect endpoints.
- **Rate limiting** configured in `config.go`.
- **Input validation** using `validator` package.
- **Error handling** in `errors.go`.

---

## 🎯 Summary

- **Setup environment variables** (`.env`)
- **Run with Docker**: `docker-compose up --build`
- **No manual migration needed** (handled automatically in Docker)
- **Test using Postman**
- **Check logs** using `docker logs -f app`



