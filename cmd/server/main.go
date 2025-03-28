package main

import (
	"context"
	"fmt"
	"github.com/alhaos/service/proto"
	"github.com/golang-jwt/jwt/v4"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log"
	"net"
	"sync"
	"time"
)

const (
	secretKey     = "your-256-bit-secret"
	tokenDuration = 15 * time.Minute
)

type User struct {
	Username string
	Password string
}

type InMemoryStore struct {
	users map[string]*User
}

func NewInMemoryUserStore() *InMemoryStore {
	return &InMemoryStore{users: map[string]*User{
		"admin": {
			"Admin",
			"password",
		},
	}}
}

// FindUser in memory store
func (s *InMemoryStore) FindUser(username string) (*User, error) {
	user, exist := s.users[username]
	if !exist {
		return nil, fmt.Errorf("user %s not found", username)
	}
	return user, nil
}

type JWTManager struct {
	secretKey     string
	tokenDuration time.Duration
}

func NewJWTManager(secretKey string, tokenDuration time.Duration) *JWTManager {
	return &JWTManager{
		secretKey:     secretKey,
		tokenDuration: tokenDuration,
	}
}

func (m *JWTManager) Generate(user *User) (string, error) {
	claims := jwt.MapClaims{
		"username": user.Username,
		"exp":      time.Now().Add(m.tokenDuration).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(m.secretKey))
}

func (m *JWTManager) Verify(accessToken string) (*jwt.MapClaims, error) {

	token, err := jwt.ParseWithClaims(
		accessToken,
		&jwt.MapClaims{},
		func(token *jwt.Token) (interface{}, error) {
			_, ok := token.Method.(*jwt.SigningMethodHMAC)
			if !ok {
				return nil, status.Errorf(codes.Unauthenticated, "unexpected token signing method")
			}
			return []byte(m.secretKey), nil
		},
	)

	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
	}

	claims, ok := token.Claims.(*jwt.MapClaims)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "invalid token claims")
	}

	return claims, nil
}

type UserStore interface {
	FindUser(username string) (*User, error)
}

type CounterServer struct {
	proto.UnimplementedCounterServer
	mu         sync.Mutex
	value      int32
	userStore  UserStore
	jwtManager *JWTManager
}

func (s *CounterServer) Login(_ context.Context, req *proto.LoginRequest) (*proto.LoginResponse, error) {

	user, err := s.userStore.FindUser(req.Username)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "user not found")
	}

	if user.Password != req.Password {
		return nil, status.Errorf(codes.Unauthenticated, "invalid credentials")
	}

	token, err := s.jwtManager.Generate(user)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot generate token")
	}

	return &proto.LoginResponse{Token: token}, nil
}

func (c *CounterServer) Increment(context.Context, *proto.IncrementRequest) (*proto.IncrementResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.value++
	fmt.Println(c.value)
	return &proto.IncrementResponse{CurrentValue: c.value}, nil
}

func main() {

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	userStore := NewInMemoryUserStore()
	jwtManager := NewJWTManager(secretKey, tokenDuration)

	s := grpc.NewServer()
	proto.RegisterCounterServer(s, &CounterServer{
		userStore:  userStore,
		jwtManager: jwtManager,
	})

	log.Printf("It's work!!!---!!!")
	log.Println("Server started on port 50051")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
