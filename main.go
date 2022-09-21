package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/aws/aws-lambda-go/lambda"
	_ "github.com/lib/pq"
)

const (
	// deepcode ignore HardcodedPassword:
	port = 5432
	// initialDbName = "postgres"
)

type LambdaEvent struct {
	RDSEndpoint string `json:"rds_endpoint"`
	AppName     string `json:"app_name"`
	DbUsername  string `json:"db_username"`
	DbPassword  string `json:"db_password"`
}

func HandleRequest(ctx context.Context, event LambdaEvent) (string, error) {

	// log.Printf("Received event: %s", event)

	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=require connect_timeout=5",
		event.RDSEndpoint, port, event.DbUsername, event.DbPassword, event.AppName)
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	sqlStatement := `CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
	CREATE EXTENSION IF NOT EXISTS pg_stat_statements;`
	_, err = db.Exec(sqlStatement)
	if err != nil {
		panic(err)
	}
	log.Println("Extensions created!")

	// sqlStatement = fmt.Sprintf("SELECT from pg_database WHERE datname='%s';", event.AppName)
	// out, err := db.Exec(sqlStatement)
	// if err != nil {
	// 	panic(err)
	// }
	// if x, _ := out.RowsAffected(); x > 0 {
	// 	log.Println("Database already exists!")
	// } else {
	// 	sqlStatement = fmt.Sprintf("CREATE DATABASE %s;", event.AppName)
	// 	_, err = db.Exec(sqlStatement)
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	log.Println("Database created!")
	// }

	sqlStatement = fmt.Sprintf("SELECT from pg_roles WHERE rolname='%s';", event.AppName)
	out, err := db.Exec(sqlStatement)
	if err != nil {
		panic(err)
	}
	if x, _ := out.RowsAffected(); x > 0 {
		log.Println("Service role already exists!")
	} else {
		sqlStatement = fmt.Sprintf("CREATE ROLE %s WITH LOGIN PASSWORD 'xxxxxxxxxxxxxxx';", // deepcode ignore HardcodedPassword: Vault will rotate it
			event.AppName)
		_, err = db.Exec(sqlStatement)
		if err != nil {
			panic(err)
		}
		log.Println("Service role created!")
	}

	sqlStatement = "SELECT from pg_roles WHERE rolname='read_only_user';"
	out, err = db.Exec(sqlStatement)
	if err != nil {
		panic(err)
	}
	if x, _ := out.RowsAffected(); x > 0 {
		log.Println("Read-only role already exists!")
	} else {
		sqlStatement = "CREATE ROLE read_only_user WITH LOGIN PASSWORD 'xxxxxxxxxxxxxxx';" // deepcode ignore HardcodedPassword: Vault will rotate it
		_, err = db.Exec(sqlStatement)
		if err != nil {
			panic(err)
		}
		log.Println("Read-only role created!")
	}
	sqlStatement = fmt.Sprintf("REVOKE ALL ON DATABASE %s FROM public; REVOKE ALL ON schema public FROM public;", event.AppName)
	_, err = db.Exec(sqlStatement)
	if err != nil {
		panic(err)
	}
	sqlStatement = fmt.Sprintf("GRANT ALL PRIVILEGES ON DATABASE %s TO %s; GRANT ALL ON schema public TO %s;", event.AppName, event.AppName, event.AppName)
	_, err = db.Exec(sqlStatement)
	if err != nil {
		panic(err)
	}
	sqlStatement = fmt.Sprintf(`GRANT CONNECT ON DATABASE %s to read_only_user;
	GRANT USAGE ON SCHEMA public TO read_only_user;
	GRANT SELECT ON ALL TABLES IN SCHEMA public TO read_only_user;`, event.AppName)
	_, err = db.Exec(sqlStatement)
	if err != nil {
		panic(err)
	}
	sqlStatement = fmt.Sprintf("GRANT %s to %s; ALTER DEFAULT PRIVILEGES FOR ROLE %s IN SCHEMA public GRANT SELECT ON TABLES TO read_only_user;", event.AppName, event.DbUsername, event.AppName)
	_, err = db.Exec(sqlStatement)
	if err != nil {
		panic(err)
	}
	log.Println("All GRANTS configured!")

	return "Database initialized!", nil
}

func main() {
	lambda.Start(HandleRequest)
}
