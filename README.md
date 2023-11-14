# Sports Odds Service

## Description
This Go application provides a web service for managing and querying NBA odds data. It's designed to interact with a DynamoDB table and provides several endpoints for different functionalities.

## Features
- **DynamoDB Integration**: Connects to an AWS DynamoDB table to store and retrieve NBA odds data.
- **RESTful API Design**: Includes endpoints for fetching all records, querying specific records based on different criteria, and checking the service status.
- **Request Logging**: Integrates with Loggly for logging requests.
- **Input Validation**: Validates query parameters using the `validator` package.
- **Error Handling**: Includes comprehensive error handling and logging.

## Endpoints
- `GET /jalcanta/all`: Retrieves all records from the DynamoDB table.
- `GET /jalcanta/status`: Provides the current status of the DynamoDB table, including item count.
- `GET /jalcanta/search`: Enables searching for records using an `id`, `key`, or both. This endpoint is versatile, catering to different query needs.

## Dependencies
- github.com/gorilla/mux: Handles routing of HTTP requests.
- github.com/jamespearly/loggly: Integrates with Loggly for logging purposes.
- github.com/aws/aws-sdk-go: Manages interactions with AWS services.
- github.com/go-playground/validator/v10: Validates the integrity of request parameters.
