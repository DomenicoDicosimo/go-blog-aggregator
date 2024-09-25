# Go Blog Aggregator

A robust RSS feed aggregator built with Go, featuring user authentication, feed management, and post collection.

## Table of Contents
- [Description](#description)
- [Features](#features)
- [Technology Stack](#technology-stack)
- [Getting Started](#getting-started)
  - [Prerequisites](#prerequisites)
  - [Installation](#installation)
- [Usage](#usage)
- [API Endpoints](#api-endpoints)
- [Testing](#testing)
- [Deployment](#deployment)
- [Contributing](#contributing)
- [License](#license)

## Description

Bloggo is a powerful RSS feed aggregator service that allows users to manage and follow their favorite blog feeds. It automatically collects and stores posts from followed feeds, providing a centralized platform for users to stay updated with their preferred content.

## Features

- ✅ User registration and authentication
- 📊 Feed management (create, read, follow, unfollow)
- 🔄 Automatic post collection from feeds
- 🌐 RESTful API for interacting with feeds and posts
- 🛡️ Rate limiting and CORS support
- 📦 Database migrations using Goose
- 🐳 Dockerized application for easy deployment
- 🚀 Continuous Integration/Continuous Deployment (CI/CD) pipeline

## Technology Stack

- 🖥️ Go 1.22
- 🐘 PostgreSQL 14
- 🐳 Docker & Docker Compose
- 🔧 GitHub Actions (CI/CD)
- 📚 Various Go packages including:
  - `github.com/lib/pq` for PostgreSQL driver
  - `github.com/joho/godotenv` for environment variable management
  - `github.com/julienschmidt/httprouter` for HTTP routing
  - `golang.org/x/crypto` for password hashing
  - `github.com/go-mail/mail/v2` for email sending

## Getting Started

### Prerequisites

- Go 1.22 or later
- Docker and Docker Compose
- PostgreSQL 14 (if running locally without Docker)

### Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/go-blog-aggregator.git
   cd go-blog-aggregator

2. Create a .env file in the root directory and add the necessary environment variables:
    ```bash
    PORT=8080
    DB=postgres://postgres:postgres@db:5432/blogaggregator?sslmode=disable
    MAILER_HOST=your_smtp_host
    MAILER_PORT=your_smtp_port
    MAILER_USERNAME=your_smtp_username
    MAILER_PASSWORD=your_smtp_password
    MAILER_SENDER=your_sender_email
    LIMITER_ENABLED=true
    LIMITER_RPS=2
    LIMITER_BURST=4
    TRUSTED_ORIGINS=
    ```

3. Build and start the application using Make:
    ```bash
    make run
    ```

### Usage

Once the application is running, you can interact with it through its RESTful API. Use tools like cURL or Postman to send requests to http://localhost:8080.

### API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/v1/healthz` | Healthcheck |
| POST | `/v1/users` | Create a new user |
| PUT | `/v1/users/activated` | Activate a user account |
| POST | `/v1/tokens/authentication` | Create an authentication token |
| POST | `/v1/feeds` | Create a new feed |
| GET | `/v1/feeds` | Get all feeds |
| POST | `/v1/feed_follows` | Follow a feed |
| DELETE | `/v1/feed_follows/:feedfollowID` | Unfollow a feed |
| GET | `/v1/feed_follows` | Get all followed feeds |
| GET | `/v1/posts` | Get posts from followed feeds |
| GET | `/debug/vars` | Expvar handler (for debugging) |

### Testing

    ```bash
    make test
    ```

### Deployment

The project includes a CI/CD pipeline using GitHub Actions. On push to the main branch, it will:

- 🧪 Run tests
- 🎨 Check code formatting
- 🏗️ Build and push a Docker image to GitHub Container Registry
- 🚀 Deploy to a Digital Ocean droplet

For manual deployment, you can use:
    ```bash
    make docker-build
    make docker-run
    ```

### Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
License
This project is open source and available under the MIT License.