package main
import (
	"encoding/json"
	"log"
	"os"
	"net/http"
	"github.com/gorilla/mux"
	"github.com/jamespearly/loggly"
	"github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/dynamodb"
    "github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/go-playground/validator/v10"
)

type Outcome struct {
	Name  string `json:"name"`
	Price string `json:"price"`
}

type Item struct {
	ID       string    `json:"id"`
	Key      string    `json:"key"`
	Outcomes []Outcome `json:"outcomes"`
}

// Struct for the log entry
type LogEntry struct {
	Method   string `json:"method"`
	SourceIP string `json:"sourceIP"`
	Path     string `json:"path"`
	Status   int    `json:"status"`
}

// Define the SearchParams struct with validation rules
type SearchParams struct {
    ID  string `validate:"omitempty,alphanum,max=100"` // ID should be alphanumeric, up to 100 characters
    Key string `validate:"omitempty,alpha,max=50"`      // Key should be alphabetic, up to 50 characters
}

// custom response writer to capture the HTTP status code
type responseWriter struct {
	http.ResponseWriter
	status int
}

func allHandler(w http.ResponseWriter, req *http.Request) {
    sess, err := session.NewSession(&aws.Config{
        Region: aws.String(os.Getenv("AWS_DEFAULT_REGION")),
    })
    if err != nil {
        log.Fatalf("Failed to create AWS session: %s", err)
    }

    // Create DynamoDB client
    svc := dynamodb.New(sess)

    // Define the Scan input parameters
    params := &dynamodb.ScanInput{
        TableName: aws.String("nba-odds-jalcanta"),
    }

	    // Perform the Scan operation and get the result
    result, err := svc.Scan(params)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    // Define a slice of the 'Item' struct type to hold the unmarshalled results
    var items []Item

    // Unmarshal the results into the slice of 'Item' structs
    err = dynamodbattribute.UnmarshalListOfMaps(result.Items, &items)
    if err != nil {
        http.Error(w, "Failed to unmarshal DynamoDB scan items", http.StatusInternalServerError)
        return
    }

	// Set the Content-Type header
    w.Header().Set("Content-Type", "application/json")

	    // Write the JSON response
    err = json.NewEncoder(w).Encode(items)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

}


// responds with the system's current time
func statusHandler(w http.ResponseWriter, req *http.Request) {
    // Define the table name
    tableName := "nba-odds-jalcanta"

	svc := getDynamoDBClient()

	    // Define the input parameters for the DescribeTable operation
    input := &dynamodb.DescribeTableInput{
        TableName: aws.String(tableName),
    }

	    // Perform the DescribeTable operation
    result, err := svc.DescribeTable(input)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

	    // Extract the count of items from the result
    itemCount := *result.Table.ItemCount

    // Set the response object
    response := map[string]interface{}{
        "table":       tableName,
        "recordCount": itemCount,
    }

    // Set the Content-Type header
    w.Header().Set("Content-Type", "application/json")

    // Write the JSON response
    w.WriteHeader(http.StatusOK)
    err = json.NewEncoder(w).Encode(response)
    if err != nil {
        http.Error(w, "Failed to encode response to JSON", http.StatusInternalServerError)
    }
}

// Create a validator instance
var validate *validator.Validate

func init() {
    validate = validator.New()
}

func searchHandler(w http.ResponseWriter, req *http.Request) {

   svc := getDynamoDBClient()

   // Get 'id' and 'key' from query parameters
   id := req.URL.Query().Get("id")
   key := req.URL.Query().Get("key")

   searchParams := SearchParams{
	ID:  req.URL.Query().Get("id"),
	Key: req.URL.Query().Get("key"),
}

   // Validate the searchParams
   err := validate.Struct(searchParams)
   if err != nil {
	   // Return a 400 Bad Request with the validation error message
	   http.Error(w, "400 Bad Request", http.StatusBadRequest)
	   return
   }

   // Variables to store the results and any potential error
   var items []Item

   // If 'id' is provided, perform a Query operation
   if id != "" {
	   exprAttrNames := map[string]*string{"#id": aws.String("id")}
	   exprAttrValues := map[string]*dynamodb.AttributeValue{
		   ":idValue": {S: aws.String(id)},
	   }
	   keyCondExpr := "#id = :idValue"

	   // Add Key condition if 'key' is also provided
	   if key != "" {
		   exprAttrNames["#key"] = aws.String("key")
		   exprAttrValues[":keyValue"] = &dynamodb.AttributeValue{S: aws.String(key)}
		   keyCondExpr += " AND #key = :keyValue"
	   }

	   queryInput := &dynamodb.QueryInput{
		   TableName:                 aws.String("nba-odds-jalcanta"),
		   ExpressionAttributeNames:  exprAttrNames,
		   ExpressionAttributeValues: exprAttrValues,
		   KeyConditionExpression:    &keyCondExpr,
	   }

	   // Perform the query operation on DynamoDB
	   var queryResult *dynamodb.QueryOutput
	   queryResult, err = svc.Query(queryInput)
	   if err != nil {
		   http.Error(w, err.Error(), http.StatusInternalServerError)
		   return
	   }

	   // If the query returns no results, return a 404 Not Found
	   if len(queryResult.Items) == 0 {
		   http.Error(w, "404 Page Not Found", http.StatusNotFound)
		   return
	   }

	   // Unmarshal the result set into items
	   if unmarshalErr := dynamodbattribute.UnmarshalListOfMaps(queryResult.Items, &items); unmarshalErr != nil {
		   http.Error(w, unmarshalErr.Error(), http.StatusInternalServerError)
		   return
	   }

	   w.Header().Set("Content-Type", "application/json")
	   w.WriteHeader(http.StatusOK)

    } else if key != "" {
        // If only 'key' is provided, perform a Scan operation
        scanInput := &dynamodb.ScanInput{
            TableName: aws.String("nba-odds-jalcanta"),
            FilterExpression: aws.String("#key = :keyValue"),
            ExpressionAttributeNames: map[string]*string{"#key": aws.String("key")},
            ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{":keyValue": {S: aws.String(key)}},
        }

        // Perform the scan operation on DynamoDB
        var scanResult *dynamodb.ScanOutput
        scanResult, err = svc.Scan(scanInput)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }

		// Check if the scan returns no results and return a 404 Not Found if true
        if len(scanResult.Items) == 0 {
            http.Error(w, "404 Page Not Found", http.StatusNotFound)
            return
        }

        if unmarshalErr := dynamodbattribute.UnmarshalListOfMaps(scanResult.Items, &items); unmarshalErr != nil {
            http.Error(w, unmarshalErr.Error(), http.StatusInternalServerError)
            return
        }
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

    } else {
        // If neither 'id' nor 'key' is provided, return an error message
        http.Error(w, "400 Bad Request", http.StatusBadRequest)
        return
    }

    // Encode items to JSON and handle errors
    if err := json.NewEncoder(w).Encode(items); err != nil {
        http.Error(w, "Failed to encode items to JSON", http.StatusInternalServerError)
    }
}



func catchAll(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        res := &responseWriter{ResponseWriter: w} // Initialize the custom response writer to capture the status code
        defer func() {
            if err := recover(); err != nil {
                log.Printf("Recovered from a panic: %v", err)
                http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
            }
            // Log the request and response after the handler has finished executing
            logEntry := LogEntry{
                Method:   r.Method,
                SourceIP: r.RemoteAddr,
                Path:     r.URL.Path,
                Status:   res.status,
            }
            logData, err := json.Marshal(logEntry)
            if err != nil {
                log.Printf("Failed to marshal logEntry: %v", err)
            } else {
                var tag string = "loggly-agent"
                client := loggly.New(tag)
                client.Send("info", string(logData))
            }
        }()
        next.ServeHTTP(res, r) // Call the next handler
    })
}


// // WriteHeader captures the status code before writing it
func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

func getDynamoDBClient() *dynamodb.DynamoDB {
    if awsSession == nil {
		var err error
		awsSession, err = session.NewSession(&aws.Config{
			Region: aws.String(os.Getenv("AWS_DEFAULT_REGION")),
		})
		if err != nil {
			log.Fatalf("Failed to create AWS session: %s", err)
		}
	}
    return dynamodb.New(awsSession)
}

var awsSession *session.Session

func main() {

	// Set up the router
	r := mux.NewRouter()
	
	// Route for all endpoint
	r.HandleFunc("/jalcanta/all", allHandler).Methods("GET")

	// Define the route for "jalcanta/status"
	r.HandleFunc("/jalcanta/status", statusHandler).Methods("GET")

	// Define the route for the search endpoint
	r.HandleFunc("/jalcanta/search", searchHandler).Methods("GET")

	r.Use(catchAll)

	// Start server on port 8080
	log.Fatal(http.ListenAndServe(":8080", r))
}