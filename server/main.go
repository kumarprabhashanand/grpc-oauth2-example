package main

import (
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"grpc-oauth2-example-server/pb"
	"log"
	"net"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/examples/data"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/dgrijalva/jwt-go"
	"github.com/lestrrat-go/jwx/jwk"
)

var port = flag.Int("port", 9091, "the port to serve on")

const jwksURL = `http://localhost:9090/oauth2/jwks`

var (
	errMissingMetadata = status.Errorf(codes.InvalidArgument, "missing metadata")
	errInvalidToken    = status.Errorf(codes.Unauthenticated, "invalid token")
)

type server struct {
	pb.UnimplementedGreetUserServiceServer
}

func (s *server) GreetUser(ctx context.Context, in *pb.GreetUserRequest) (*pb.GreetUserResponse, error) {
	log.Printf("Received: %v", in.GetUser())
	return &pb.GreetUserResponse{GreetMessage: "Hello " + in.GetUser().GetFirstName()}, nil
}

func main() {
	fmt.Println("main method")
	flag.Parse()
	fmt.Printf("server starting on port %d...\n", *port)
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	cert, err := tls.LoadX509KeyPair(data.Path("x509/server_cert.pem"), data.Path("x509/server_key.pem"))
	if err != nil {
		log.Fatalf("failed to load key pair: %s", err)
	}

	log.Printf("Fetching Token")
	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(ensureValidToken),
		grpc.Creds(credentials.NewServerTLSFromCert(&cert)),
	}

	s := grpc.NewServer(opts...)
	pb.RegisterGreetUserServiceServer(s, &server{})
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

}

func ensureValidToken(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errMissingMetadata
	}
	if !valid(md["authorization"]) {
		return nil, errInvalidToken
	}
	return handler(ctx, req)
}

func valid(authorization []string) bool {
	if len(authorization) < 1 {
		return false
	}
	token := strings.TrimPrefix(authorization[0], "Bearer ")
	t, e := jwt.Parse(token, getKey)

	if e != nil {
		log.Panic(e)
		return false
	}
	claims := t.Claims.(jwt.MapClaims)
	hasValidScope := false
	for key, value := range claims {
		//fmt.Printf("%s\t%v\n", key, value)
		if key == "scope" && strings.Contains(value.([]interface{})[0].(string), "articles.read") {
			hasValidScope = true
		}
	}
	return t.Valid && hasValidScope
}

func getKey(token *jwt.Token) (interface{}, error) {
	ctx := context.Background()
	set, err := jwk.Fetch(ctx, jwksURL)
	if err != nil {
		return nil, err
	}
	keyID, ok := token.Header["kid"].(string)
	if !ok {
		return nil, errors.New("expecting JWT header to have string kid")
	}
	key, flag := set.LookupKeyID(keyID)
	if flag == true {
		var pubkey interface{}
		e := key.Raw(&pubkey)

		if e == nil {
			return pubkey, nil
		}
		log.Printf("Got error in key fetching %v", e)
	}
	return nil, fmt.Errorf("unable to find key %q", keyID)
}

//https://stackoverflow.com/questions/41077953/how-to-verify-jwt-signature-with-jwk-in-go
//https://docs.spring.io/spring-authorization-server/docs/current/reference/html/configuration-model.html#configuring-authorization-server-settings
//https://docs.spring.io/spring-authorization-server/docs/current/reference/html/overview.html
//https://pkg.go.dev/golang.org/x/oauth2
