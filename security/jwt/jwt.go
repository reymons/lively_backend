package jwt

type Service interface {
	AccessToken() Token

	RefreshToken() Token
}

type jwtService struct {
	accessToken  Token
	refreshToken Token
}

func NewService(accessToken, refreshToken Token) Service {
	return &jwtService{accessToken, refreshToken}
}

func (s *jwtService) AccessToken() Token {
	return s.accessToken
}

func (s *jwtService) RefreshToken() Token {
	return s.refreshToken
}
