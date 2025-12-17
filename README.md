# Rotten Bikes

An application for managing (bicing) bike reviews and ratings.

## Getting Started

### Prerequisites

*   **Go**: Version 1.23 or higher.
*   **PostgreSQL**: A running PostgreSQL instance.
*   **Golang Migrate**: CLI tool for database migrations.
*   **Make**: For running Makefile commands.

### Installation & Run

1.  **Start the local database:**
    This command uses a helper script to start a Postgres container (requires Docker).
    ```bash
    make db-up
    ```

2.  **Run database migrations:**
    Apply the schema to the database.
    ```bash
    make db-migrate-up
    ```

3.  **Start the API server:**
    ```bash
    make run
    ```
    The server will start on `localhost:8080`.

### Database Reset
To drop the database and re-apply all migrations (fresh start):
```bash
make db-reset
```

## API Endpoints

### Authentication
| Method | Endpoint | Description | Auth Required |
| :--- | :--- | :--- | :--- |
| `POST` | `/auth/request-magic-link` | Request a magic link for login. | No |
| `POST` | `/auth/confirm` | Confirm magic link and receive Bearer token. | No |

### Bikes
| Method | Endpoint | Description | Auth Required |
| :--- | :--- | :--- | :--- |
| `GET` | `/bikes` | List all bikes. | No |
| `POST` | `/bikes` | Create a new bike. | **Yes** |
| `GET` | `/bikes/{id}` | Get details of a specific bike. | No |
| `PUT` | `/bikes/{id}` | Update a specific bike. | **Yes** |
| `DELETE` | `/bikes/{id}` | Delete a specific bike. | **Yes** |

### Bike Sub-resources
| Method | Endpoint | Description | Auth Required |
| :--- | :--- | :--- | :--- |
| `GET` | `/bikes/{id}/reviews` | Get all reviews for a specific bike. | No |
| `POST` | `/bikes/{id}/reviews` | Create a review for a specific bike. | **Yes** |
| `GET` | `/bikes/{id}/ratings` | Get rating aggregates for a specific bike. | No |
| `GET` | `/bikes/reviews` | List all reviews across all bikes. | No |
| `GET` | `/bikes/ratings` | List all rating aggregates across all bikes. | No |

### Reviews
| Method | Endpoint | Description | Auth Required |
| :--- | :--- | :--- | :--- |
| `GET` | `/reviews/{id}` | Get a specific review. | No |
| `PUT` | `/reviews/{id}` | Update a specific review. | **Yes** |
| `DELETE` | `/reviews/{id}` | Delete a specific review. | **Yes** |

### System
| Method | Endpoint | Description | Auth Required |
| :--- | :--- | :--- | :--- |
| `GET` | `/healthz` | Health check endpoint. | No |

## Configuration

The application currently defaults to the following configuration (can be seen in `cmd/api/main.go` and `Makefile`):

*   **Port**: `8080`
*   **Database User**: `rottenbikes`
*   **Database Password**: `rottenbikes`
*   **Database Name**: `rottenbikes`
*   **Database Host**: `localhost:5432`
