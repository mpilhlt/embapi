package models

// Version constants for the API
const (
	APIVersion     = "0.0.1"
	APIName        = "DHaMPS Vector Database API"
	APIDescription = "PostgreSQL-backed vector database with pgvector support, providing a RESTful API for managing embeddings in Retrieval Augmented Generation (RAG) workflows"
)

// Options for the CLI.
type Options struct {
	Debug          bool   `                       env:"SERVICE_DEBUG"          doc:"Enable debug logging"                    short:"d" default:"true"`
	Verbose        bool   `name:"verbose"         env:"SERVICE_VERBOSE"        doc:"Enable verbose logging"                            default:"false"`
	Host           string `                       env:"SERVICE_HOST"           doc:"Hostname to listen on"                             default:"localhost"`
	Port           int    `                       env:"SERVICE_PORT"           doc:"Port to listen on"                       short:"p" default:"8880"`
	DBHost         string `name:"db-host"         env:"SERVICE_DBHOST"         doc:"Database hostname"                                 default:"localhost"`
	DBPort         int    `name:"db-port"         env:"SERVICE_DBPORT"         doc:"Database port"                                     default:"5432"`
	DBUser         string `name:"db-user"         env:"SERVICE_DBUSER"         doc:"Database username"                                 default:"postgres"`
	DBPassword     string `name:"db-password"     env:"SERVICE_DBPASSWORD"     doc:"Database password"                                 default:"password"`
	DBName         string `name:"db-name"         env:"SERVICE_DBNAME"         doc:"Database name"                                     default:"postgres"`
	AdminKey       string `name:"admin-key"       env:"SERVICE_ADMINKEY"       doc:"Admin API key"`
	APIName        string `name:"api-name"        env:"SERVICE_API_NAME"       doc:"Service name for API manifest"`
	APIDescription string `name:"api-description" env:"SERVICE_API_DESCRIPTION" doc:"Service description for API manifest"`
	APIDocURL      string `name:"api-doc-url"     env:"SERVICE_API_DOC_URL"    doc:"Documentation URL for API manifest"`
}
