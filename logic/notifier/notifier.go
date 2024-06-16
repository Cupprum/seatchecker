package main

import (
	"log"

	"github.com/aws/aws-lambda-go/lambda"
)

// Event defines your lambda input and output data structure,
// and of course you can have different input and output data structure
type InEvent struct {
	Sk string `json:"sampleKey1"`
	K  string `json:"key3"`
}

type OutEvent struct {
	Status int `json:"status"`
}

func handler(request InEvent) (OutEvent, error) {
	log.Println("Program started.")

	log.Println(request)

	// email := os.Getenv("SEATCHECKER_EMAIL")
	// if email == "" {
	// 	fmt.Fprintf(os.Stderr, "env var 'SEATCHECKER_EMAIL' is not configured")
	// 	os.Exit(1)
	// }

	// log.Println("Configuration successful.")

	// catchErr := func(err error) {
	// 	if err != nil {
	// 		fmt.Fprintf(os.Stderr, "error: %v\n", err)
	// 		os.Exit(1)
	// 	}
	// }

	// log.Printf("Start account login for user: %s.\n", email)
	// cAuth, err := client.accountLogin(email, password)
	// catchErr(err) // TODO: probably i will have to rethink exceptions -> Why?
	// log.Println("Account login finished successfully.")

	log.Println("Program finished successfully.")
	return OutEvent{Status: 200}, nil
}

func main() {
	lambda.Start(handler)
	// resp := handler(InEvent{Sk: "test1", K: "test2"})
	// log.Println(resp)
}
