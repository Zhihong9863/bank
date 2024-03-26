
# Bank Project

## Project Overview
Bank is a simulated banking backend service project that demonstrates the design, development, and deployment processes of a backend system. This project covers core banking functionalities such as account management, transaction records, and funds transfer, providing a complete API support for frontend applications or services. The project leverages modern tools and technology stacks to build, test, and deploy a secure and reliable service.

## Key Features
- **Account Management**: Supports the creation of new bank accounts and management of account information.
- **Transaction Records**: Automatically records all transaction activities for each account, ensuring the accuracy of account balances.
- **Funds Transfer**: Enables secure and fast transfer of funds between different accounts.

## Technology Stack
- **Golang**: The primary development language, chosen for its advantages in concurrent processing and system performance.
- **PostgreSQL**: Used for storing user accounts and transaction data, offering transaction support to ensure data consistency.
- **Redis**: Acts as a message queue for handling asynchronous tasks and improving system response speed.
- **Gin**: A high-performance HTTP Web framework used for building RESTful APIs.
- **gRPC**: For internal service communication, enhancing the efficiency of data exchange between systems.
- **Docker & Kubernetes**: For containerizing the application and automating deployment to the cloud platform, improving consistency across development, testing, and production environments.
- **AWS**: Cloud service provider hosting Kubernetes clusters and other resources, ensuring the service's scalability and reliability.
- **GitHub Actions**: Implements the CI/CD process for automating code build, test, and deployment.

## Development Experience

### Database Design and Implementation
Designed the database architecture from scratch to ensure the consistency and integrity of the data model. Explored the importance of database transactions and different isolation levels to avoid concurrency issues such as dirty reads, phantom reads, and non-repeatable reads.

Used Docker to quickly set up a local development environment, managed project code with Git, and implemented automated unit testing through GitHub Actions.

### Building RESTful HTTP API
Constructed a complete RESTful HTTP API using the Gin framework, learning to efficiently load application configurations, mock databases for unit testing, handle errors, and verify user identity. Implemented secure authentication mechanisms with JWT and PASETO.
![49cfb24270b80b6b85bab2566f95ec8](https://github.com/Zhihong9863/bank/assets/129224800/57b1ae12-03c7-4967-b14f-d7b3d1182903)

### Containerization and Cloud Deployment
Packaged the application into a Docker image, minimizing image size for improved deployment efficiency. Configured the production environment on AWS, including creating production databases, configuring storage, and managing production secrets with Amazon EKS. Implemented a CI/CD process for automatic image building and deployment to EKS clusters. Configured domain names and protected connections with Let's Encrypt TLS certificates.
![311fc0e51820b2dd37cd53a002b2e61](https://github.com/Zhihong9863/bank/assets/129224800/807badd7-5894-4365-aec4-d92cf716e156)

### Advanced Backend Design
Delved into advanced topics such as managing user sessions, building gRPC APIs, serving both gRPC and HTTP requests with gRPC gateways, embedding Swagger documentation into the backend service, and implementing structured logging with HTTP middleware and gRPC interceptors.
![c535bf37ab8b1d55b1d816cdbab5fdb](https://github.com/Zhihong9863/bank/assets/129224800/880220e2-a1b3-4ac3-a337-af517ca3f1cf)
![be8cb37d6afb8b148f0b7d55f046560](https://github.com/Zhihong9863/bank/assets/129224800/cc72b778-d43a-4979-8b41-153e77622126)

### Asynchronous Processing and Message Queuing
Explored asynchronous processing methods in Golang using background workers and Redis as a message queue, including sending emails via the Gmail SMTP server and writing unit tests for gRPC services, which involved mocking multiple dependencies.

### Enhancing Stability and Security
Focused on improving server stability and security, including regular dependency updates, making refresh tokens more secure with cookies, and learning how to gracefully shut down servers to protect processing resources.

## Local Development Setup

### Installing Tools

Ensure the following tools are installed on the local development machine:

- **Docker Desktop**: Essential for containerization and running services like PostgreSQL.
- **TablePlus**: A versatile tool for database management.
- **Golang**: The programming language used for development.
- **Homebrew**: Package manager for macOS, useful for installing other dependencies.

#### Development Dependencies

Install the following dependencies for a complete development setup:

- **Go Migrate**
  ```bash
  brew install golang-migrate
  ```
- **DBDocs** for database documentation
  ```bash
  npm install -g dbdocs
  dbdocs login
  ```
- **DBML CLI** for working with Database Markup Language
  ```bash
  npm install -g @dbml/cli
  dbml2sql --version
  ```
- **SQLC** for generating Go code from SQL
  ```bash
  brew install sqlc
  ```
- **GoMock** for generating mock interfaces
  ```bash
  go install github.com/golang/mock/mockgen@v1.6.0
  ```

### Setting Up Infrastructure

Follow these steps to prepare the infrastructure:

1. **Create a Network for Containers**
   ```bash
   make network
   ```
2. **Start PostgreSQL Container**
   ```bash
   make postgres
   ```
3. **Create `bank` Database**
   ```bash
   make createdb
   ```

#### Database Migration

Manage database migrations with these commands:

- **Apply All Migrations**
  ```bash
  make migrateup
  ```
- **Apply a Single Migration**
  ```bash
  make migrateup1
  ```
- **Revert All Migrations**
  ```bash
  make migratedown
  ```
- **Revert a Single Migration**
  ```bash
  make migratedown1
  ```

### Documentation

Generate and access database documentation:

- **Generate Documentation**
  ```bash
  make db_docs
  ```
  Documentation is accessible at a specific URL with the password `secret`.

### Code Generation

Use these commands to generate necessary code:

- **Generate SQL File with DBML**
  ```bash
  make db_schema
  ```
- **Generate SQL CRUD Operations with SQLC**
  ```bash
  make sqlc
  ```
- **Generate Database Mocks with GoMock**
  ```bash
  make mock
  ```
- **Create New Database Migration**
  ```bash
  make new_migration name=<migration_name>
  ```

### Running the Application

Commands to start the server and run tests:

- **Start the Server**
  ```bash
  make server
  ```
- **Run Tests**
  ```bash
  make test
  ```

### Deployment to Kubernetes Cluster

Install necessary components for Kubernetes deployment:

- **Nginx Ingress Controller**
  ```bash
  kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v0.48.1/deploy/static/provider/aws/deploy.yaml
  ```
- **Cert-Manager** for SSL certificates
  ```bash
  kubectl apply -f https://github.com/jetstack/cert-manager/releases/download/v1.4.0/cert-manager.yaml
  ```

