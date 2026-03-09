package jwt

type JWTService interface {
	AccessToken() Token

	RefreshToken() Token
}

type jwtService struct {
	accessToken  Token
	refreshToken Token
}

func NewService(accessToken, refreshToken Token) JWTService {
	return &jwtService{accessToken, refreshToken}
}

func (s *jwtService) AccessToken() Token {
	return s.accessToken
}

func (s *jwtService) RefreshToken() Token {
	return s.refreshToken
}
