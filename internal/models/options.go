package models

// Options for the CLI.
type Options struct {
	Debug       bool   `                    env:"SERVICE_DEBUG"       doc:"Enable debug logging"                    short:"d" default:"true"`
	AuthVerbose bool   `name:"auth-verbose" env:"SERVICE_AUTH_VERBOSE" doc:"Enable verbose authentication logging"  short:"v" default:"false"`
	Host        string `                    env:"SERVICE_HOST"        doc:"Hostname to listen on"                             default:"localhost"`
	Port        int    `                    env:"SERVICE_PORT"        doc:"Port to listen on"                       short:"p" default:"8880"`
	DBHost      string `name:"db-host"      env:"SERVICE_DBHOST"      doc:"Database hostname"                                 default:"localhost"`
	DBPort      int    `name:"db-port"      env:"SERVICE_DBPORT"      doc:"Database port"                                     default:"5432"`
	DBUser      string `name:"db-user"      env:"SERVICE_DBUSER"      doc:"Database username"                                 default:"postgres"`
	DBPassword  string `name:"db-password"  env:"SERVICE_DBPASSWORD"  doc:"Database password"                                 default:"password"`
	DBName      string `name:"db-name"      env:"SERVICE_DBNAME"      doc:"Database name"                                     default:"postgres"`
	AdminKey    string `name:"admin-key"    env:"SERVICE_ADMINKEY"    doc:"Admin API key"`
}
