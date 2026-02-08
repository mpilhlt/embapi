package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/mpilhlt/embapi/internal/auth"
	"github.com/mpilhlt/embapi/internal/database"
	"github.com/mpilhlt/embapi/internal/handlers"
	"github.com/mpilhlt/embapi/internal/models"

	"github.com/danielgtaylor/huma/v2/adapters/humago"
	"github.com/danielgtaylor/huma/v2/autopatch"
	"github.com/danielgtaylor/huma/v2/humacli"
	"github.com/joho/godotenv"

	huma "github.com/danielgtaylor/huma/v2"
)

func main() {
	// Load env variables
	err := godotenv.Load()
	if err != nil {
		fmt.Println("No .env file found")
	}

	// Create a CLI app
	cli := humacli.New(func(hooks humacli.Hooks, options *models.Options) {

		// workarounds for env variables
		if os.Getenv("SERVICE_DEBUG") != "" {
			options.Debug = os.Getenv("SERVICE_DEBUG") == "true"
		}
		if os.Getenv("SERVICE_HOST") != "" {
			options.Host = os.Getenv("SERVICE_HOST")
		}
		if os.Getenv("SERVICE_PORT") != "" {
			port, err := strconv.Atoi(os.Getenv("SERVICE_PORT"))
			if err == nil {
				options.Port = port
			}
		}
		if os.Getenv("SERVICE_DBHOST") != "" {
			options.DBHost = os.Getenv("SERVICE_DBHOST")
		}
		if os.Getenv("SERVICE_DBPORT") != "" {
			dbPort, err := strconv.Atoi(os.Getenv("SERVICE_DBPORT"))
			if err == nil {
				options.DBPort = dbPort
			}
		}
		if os.Getenv("SERVICE_DBNAME") != "" {
			options.DBName = os.Getenv("SERVICE_DBNAME")
		}
		if os.Getenv("SERVICE_DBUSER") != "" {
			options.DBUser = os.Getenv("SERVICE_DBUSER")
		}
		if os.Getenv("SERVICE_DBPASSWORD") != "" {
			options.DBPassword = os.Getenv("SERVICE_DBPASSWORD")
		}
		if os.Getenv("SERVICE_ADMINKEY") != "" {
			options.AdminKey = os.Getenv("SERVICE_ADMINKEY")
		}

		println()
		println("=== Starting DH@MPS Vector Database ...")
		fmt.Printf("    Options are debug:%v host:%v port: %v dbhost:%s dbname:%s\n",
			options.Debug, options.Host, options.Port, options.DBHost, options.DBName)

		// Validate required environment variables
		if os.Getenv("ENCRYPTION_KEY") == "" {
			fmt.Println("    ERROR: ENCRYPTION_KEY environment variable is not set")
			fmt.Println("    Please set ENCRYPTION_KEY in your .env file or environment")
			os.Exit(1)
		}

		// Initialize the database
		pool, err := database.InitDB(options)
		if err != nil {
			fmt.Printf("    Unable to connect to database: %v\n", err)
			os.Exit(1)
		}
		// defer pool.Close()

		// Define standard key generator (for API keys)
		keyGen := handlers.StandardKeyGen{}

		// Create a new router & API
		config := huma.DefaultConfig("DHaMPS Vector Database API", "0.0.1")
		config.Components.SecuritySchemes = auth.Config
		router := http.NewServeMux()

		// Register a global OPTIONS andler before creating the API
		router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "OPTIONS" {
				// fmt.Print("    OPTIONS request received, handled in main function.\n")
				w.Header().Set("Access-Control-Allow-Origin", "*")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH, UPDATE, QUERY")
				w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, Content-Disposition, Origin, X-Requested-With")
				w.WriteHeader(http.StatusOK)
				return
			}
		})

		api := humago.New(router, config)
		api.UseMiddleware(auth.CORSMiddleware(api))
		api.UseMiddleware(auth.EmbAPIKeyAdminAuth(api, options))
		api.UseMiddleware(auth.EmbAPIKeyOwnerAuth(api, pool, options))
		api.UseMiddleware(auth.EmbAPIKeyEditorAuth(api, pool, options))
		api.UseMiddleware(auth.EmbAPIKeyReaderAuth(api, pool, options))
		api.UseMiddleware(auth.AuthTermination(api))

		// Add routes to the API
		err = handlers.AddRoutes(pool, keyGen, api)
		if err != nil {
			fmt.Printf("    Unable to add routes: %v\n", err)
			os.Exit(1)
		}

		// Add AutoPatch to automatically create PATCH endpoints for resources with GET+PUT
		autopatch.AutoPatch(api)

		// Create the HTTP server
		server := &http.Server{
			Addr:    fmt.Sprintf("%s:%d", options.Host, options.Port),
			Handler: router,
		}

		// Start server
		hooks.OnStart(func() {
			fmt.Printf("=== Starting API server on port %d...\n\n", options.Port)
			// go func() {
			err := server.ListenAndServe()
			if err != nil && err != http.ErrServerClosed {
				fmt.Printf("listen error: %s\n", err)
			} else {
				fmt.Printf("    API server on port %d stopped.\n", options.Port)
			}
			// }()

			// Keep the program running
			// select {}
		})

		// Gracefully shutdown server
		hooks.OnStop(func() {
			fmt.Printf("\n=== Shutting down API server on port %d...\n", options.Port)

			// Create a context with a timeout for the shutdown process
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Attempt to gracefully shut down the server
			if err := server.Shutdown(ctx); err != nil {
				fmt.Printf("Shutdown error: %v\n", err)
			}

			// Close the database pool
			activeConns := pool.Stat().TotalConns()
			fmt.Printf("    Active connections before shutdown: %d\n", activeConns)

			pool.Close()
			fmt.Println("    Database pool successfully closed.")
			fmt.Print("=== DH@MPS Vector Database stopped.\n\n")
		})
	})

	// Run the CLI. When passed no commands, it starts the server.
	cli.Run()
}
