package server

import (
	"context"

	"auth-service/internal/service"
	"auth-service/pb"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)


type GRPCServer struct {
	pb.UnimplementedAuthServiceServer 
	authService *service.AuthService
}

func NewGRPCServer(authService *service.AuthService) *GRPCServer {
	return &GRPCServer{authService: authService}
}

func (s *GRPCServer) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	if req.GetEmail() == "" || req.GetPassword() == "" {
		return nil, status.Error(codes.InvalidArgument, "email and password are required")
	}

	userID, err := s.authService.Register(ctx, req.GetEmail(), req.GetPassword(), req.GetUserType().String())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "registration failed: %v", err)
	}

	return &pb.RegisterResponse{UserId: userID}, nil
}

func (s *GRPCServer) Login(ctx context.Context, req *pb.LoginRequest) (*pb.TokenResponse, error) {
	access, refresh, err := s.authService.Login(ctx, req.GetEmail(), req.GetPassword())
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}

	return &pb.TokenResponse{
		AccessToken:  access,
		RefreshToken: refresh,
	}, nil
}

func (s *GRPCServer) RefreshToken(ctx context.Context, req *pb.RefreshRequest) (*pb.TokenResponse, error) {
	if req.GetRefreshToken() == "" {
		return nil, status.Error(codes.InvalidArgument, "refresh token is required")
	}

	access, refresh, err := s.authService.RefreshToken(ctx, req.GetRefreshToken())
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}

	return &pb.TokenResponse{
		AccessToken:  access,
		RefreshToken: refresh,
	}, nil
}

func (s *GRPCServer) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	if req.GetRefreshToken() == "" {
		return nil, status.Error(codes.InvalidArgument, "refresh token is required")
	}

	err := s.authService.Logout(ctx, req.GetRefreshToken())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "logout failed: %v", err)
	}

	return &pb.LogoutResponse{Success: true}, nil
}

func (s *GRPCServer) ValidateToken(ctx context.Context, req *pb.ValidateRequest) (*pb.ValidateResponse, error) {
	if req.GetAccessToken() == "" {
		return nil, status.Error(codes.InvalidArgument, "access token is required")
	}

	// ИСПРАВЛЕНО: Принимаем groupID четвертым возвращаемым аргументом из бизнес-логики
	userID, userType, groupID, err := s.authService.ValidateToken(ctx, req.GetAccessToken())
	if err != nil {
		return &pb.ValidateResponse{IsValid: false}, status.Error(codes.Unauthenticated, "invalid token")
	}

	// ИСПРАВЛЕНО: Прокидываем GroupId в наружный protobuf ответ для Трекера Еды
	return &pb.ValidateResponse{
		UserId:   userID,
		IsValid:  true,
		UserType: userType,
		GroupId:  groupID, // Поле запишется в ValidateResponse.group_id
	}, nil
}

