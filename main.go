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
	port          = 5432
	initialDbName = "postgres"
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
		event.RDSEndpoint, port, event.DbUsername, event.DbPassword, initialDbName)
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

	sqlStatement = fmt.Sprintf("SELECT from pg_database WHERE datname='%s';", event.AppName)
	out, err := db.Exec(sqlStatement)
	if err != nil {
		panic(err)
	}
	if x, _ := out.RowsAffected(); x > 0 {
		log.Println("Database already exists!")
	} else {
		sqlStatement = fmt.Sprintf("CREATE DATABASE %s;", event.AppName)
		_, err = db.Exec(sqlStatement)
		if err != nil {
			panic(err)
		}
		log.Println("Database created!")
	}

	sqlStatement = fmt.Sprintf("SELECT from pg_roles WHERE rolname='%s';", event.AppName)
	out, err = db.Exec(sqlStatement)
	if err != nil {
		panic(err)
	}
	if x, _ := out.RowsAffected(); x > 0 {
		log.Println("Role already exists!")
	} else {
		sqlStatement = fmt.Sprintf("CREATE ROLE %s WITH LOGIN PASSWORD 'xxxxxxxxxxxxxxx';", // deepcode ignore HardcodedPassword: Vault will rotate it
			event.AppName)
		_, err = db.Exec(sqlStatement)
		if err != nil {
			panic(err)
		}
		log.Println("Role created!")
	}

	sqlStatement = fmt.Sprintf("GRANT ALL PRIVILEGES ON DATABASE \"%s\" TO \"%s\";", event.AppName, event.AppName)
	_, err = db.Exec(sqlStatement)
	if err != nil {
		panic(err)
	}
	log.Println("Privileges granted!")

	return "Database initialized!", nil
}

func main() {
	lambda.Start(HandleRequest)
}
