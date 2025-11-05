# ceramicraft-order-mservice
# 🚀 [ceramicraft-order-mservice]: transaction and order management system

![Go Version](https://img.shields.io/badge/go-1.24.9-blue.svg)
![Build Status](https://img.shields.io/badge/build-passing-brightgreen.svg)
![License](https://img.shields.io/badge/license-MIT-green.svg)

## 💡 Overview

This service is part of the **cerami-craft** project, responsible for order-related APIs.

### Key Features

* **Communication:** High-efficiency gRPC for internal services.
* **Discovery:**  `docker network/k8s service` depending on the final deployment form.
* **Asynchronicity:** Event-driven processing using `Kafka`.
* **Observability:** `Prometheus` & `loki`.

---

## 🏛️ Architecture & Stack

### Technology Stack

| Category | Technology | Purpose |
| :--- | :--- | :--- |
| **Language** | Go (Golang) | High concurrency and performance |
| **Framework** | `Gin, gRPC` | API and RPC handling |
| **ORM** | `gorm` | database CRUD implementation |
| **Database** | `MySQL` | Persistent storage |
| **Messaging** | `Kafka` | Event communication |

## 📂 Repository Structure

| Directory | Core Functionality | 
| :--- | :--- | 
| **`.github/workflows`** | **CI/CD:** Contains GitHub Actions configurations for automated testing, linting, and deployment |
| **`client`** | **RPC/API Stubs:** Holds the generated Go client stub code and interfaces required by other microservices to communicate with this service. |
| **`server`** | **Service Implementation:** Contains the main entry points (`main.go`) and the core business logic for the HTTP/gRPC server implementations. |
| **`common`** | **Shared Resources:** Packages containing shared data structures (e.g., Protobuf message definitions, domain models) and generic utility methods used by both `client` and `server`. |


---

## ⚙️ Getting Started

### Prerequisites

* Go `[1.24.9]`
* docker compose

### Deployment with Docker Compose

The recommended way to run the entire system (services + infrastructure) is using Docker Compose.

1.  **Clone the repository:**
    ```bash
    git clone git@github.com:NUS-ISS-Agile-Team/ceramicraft-deploy.git
    cd ceramicraft-deploy
    ```

2.  **Build and run services:**
    ```bash
    docker-compose up --build -d
    ```

    *The Swagger will be available at `http://localhost/order-ms/v1/swagger/index.html`.*