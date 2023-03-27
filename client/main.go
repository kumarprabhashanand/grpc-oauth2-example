package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc/examples/data"

	"grpc-oauth2-example-client/pb"

	"google.golang.org/grpc"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/oauth"
)

func main() {
	fmt.Println("main method")
	serverAddress := flag.String("address", "localhost:9091", "the server address")
	flag.Parse()
	log.Printf("Connecting to %s", *serverAddress)

	log.Printf("Fetching Token")
	perRPC := oauth.TokenSource{TokenSource: oauth2.StaticTokenSource(fetchToken())}
	creds, err := credentials.NewClientTLSFromFile(data.Path("x509/ca_cert.pem"), "x.test.example.com")
	if err != nil {
		log.Fatalf("failed to load credentials: %v", err)
	}
	opts := []grpc.DialOption{
		grpc.WithPerRPCCredentials(perRPC),
		grpc.WithTransportCredentials(creds),
	}

	//conn, err := grpc.Dial(*serverAddress, grpc.WithInsecure())
	conn, err := grpc.Dial(*serverAddress, opts...)
	if err != nil {
		log.Fatalf("error while connecting: %v", err)
	}
	defer conn.Close()

	client := pb.NewGreetUserServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	callGreetUser(client, ctx)

}

func callGreetUser(client pb.GreetUserServiceClient, ctx context.Context) {
	lastName := flag.String("lastName", "Doe", "last name of user")
	var age uint32 = 26

	user := pb.User{FirstName: "John", LastName: lastName, Age: &age}

	req := pb.GreetUserRequest{User: &user}

	r, err := client.GreetUser(ctx, &req)

	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}

	log.Printf("Response From Server - Greeting : %s", r.GetGreetMessage())
}

//not used right now
func fetchStaticToken() *oauth2.Token {
	return &oauth2.Token{
		AccessToken: "some-secret-token",
	}
}

func fetchToken() *oauth2.Token {
	ctx := context.Background()
	conf := &clientcredentials.Config{
		ClientID:     "oauth-client",
		ClientSecret: "oauth-secret",
		Scopes:       []string{"articles.read"},
		TokenURL:     "http://localhost:9090/oauth2/token",
	}
	token, error := conf.Token(ctx)

	if error != nil {
		log.Fatal("Error fetching access token")
	}
	//log.Printf("Got token : %s", token.AccessToken)
	return token

}

//not used right now
func fetchTokenSource() oauth2.TokenSource {
	ctx := context.Background()
	conf := &clientcredentials.Config{
		ClientID:     "oauth-client",
		ClientSecret: "oauth-secret",
		Scopes:       []string{"articles.read"},
		TokenURL:     "http://localhost:9090/oauth2/token",
	}
	tokenSource := conf.TokenSource(ctx)
	return tokenSource
}
