package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	client "github.com/ory/kratos-client-go"
)

// Kratos Client
type KratosClient struct {
	admin  *client.APIClient
	public *client.APIClient
}

type KratosUser struct {
	ID               string      `json:"id"`
	Email            string      `json:"email"`
	Username         string      `json:"username"`
	FirstName        string      `json:"first_name"`
	LastName         string      `json:"last_name"`
	Phone            string      `json:"phone"`
	SubscriptionPlan string      `json:"subscription_plan"`
	Avatar           string      `json:"avatar"`
	EmailVerified    bool        `json:"email_verified"`
	Active           bool        `json:"active"`
	CreatedAt        time.Time   `json:"created_at"`
	UpdatedAt        time.Time   `json:"updated_at"`
	Traits           interface{} `json:"traits"`
}

func NewKratosClient(adminURL, publicURL string) *KratosClient {
	adminConfig := client.NewConfiguration()
	adminConfig.Servers = []client.ServerConfiguration{
		{URL: adminURL, Description: "Kratos Admin API"},
	}

	publicConfig := client.NewConfiguration()
	publicConfig.Servers = []client.ServerConfiguration{
		{URL: publicURL, Description: "Kratos Public API"},
	}

	return &KratosClient{
		admin:  client.NewAPIClient(adminConfig),
		public: client.NewAPIClient(publicConfig),
	}
}

// Session Validation
func (k *KratosClient) ValidateSession(ctx context.Context, sessionToken string) (*KratosUser, error) {
	if sessionToken == "" {
		return nil, fmt.Errorf("session token is required")
	}

	// Проверяем сессию через Kratos
	session, resp, err := k.public.FrontendAPI.ToSession(ctx).
		XSessionToken(sessionToken).
		Execute()

	if err != nil {
		return nil, fmt.Errorf("session validation failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("invalid session: status %d", resp.StatusCode)
	}

	if !session.GetActive() {
		return nil, fmt.Errorf("session is not active")
	}

	// Преобразуем в наш формат
	user := &KratosUser{
		ID:        session.Identity.GetId(),
		Active:    session.GetActive(),
		CreatedAt: session.Identity.GetCreatedAt(),
		UpdatedAt: session.Identity.GetUpdatedAt(),
	}

	// Извлекаем traits безопасно
	if traits := session.Identity.GetTraits(); traits != nil {
		user.Traits = traits
		user.Email = getStringFromTraits(traits, "email")
		user.Username = getStringFromTraits(traits, "username")
		user.FirstName = getStringFromTraits(traits, "first_name")
		user.LastName = getStringFromTraits(traits, "last_name")
		user.Phone = getStringFromTraits(traits, "phone")
		user.SubscriptionPlan = getStringFromTraits(traits, "subscription_plan")
		user.Avatar = getStringFromTraits(traits, "avatar")
	}

	// Проверяем верификацию email через recovery addresses
	user.EmailVerified = k.isEmailVerified(ctx, session.Identity.GetId())

	return user, nil
}

// User Management
func (k *KratosClient) GetUser(ctx context.Context, userID string) (*KratosUser, error) {
	identity, resp, err := k.admin.IdentityAPI.GetIdentity(ctx, userID).
		Execute()

	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("user not found: status %d", resp.StatusCode)
	}

	user := &KratosUser{
		ID:        identity.GetId(),
		CreatedAt: identity.GetCreatedAt(),
		UpdatedAt: identity.GetUpdatedAt(),
	}

	// Извлекаем traits безопасно
	if traits := identity.GetTraits(); traits != nil {
		user.Traits = traits
		user.Email = getStringFromTraits(traits, "email")
		user.Username = getStringFromTraits(traits, "username")
		user.FirstName = getStringFromTraits(traits, "first_name")
		user.LastName = getStringFromTraits(traits, "last_name")
		user.Phone = getStringFromTraits(traits, "phone")
		user.SubscriptionPlan = getStringFromTraits(traits, "subscription_plan")
		user.Avatar = getStringFromTraits(traits, "avatar")
	}

	user.EmailVerified = k.isEmailVerified(ctx, identity.GetId())

	return user, nil
}

func (k *KratosClient) UpdateUser(ctx context.Context, userID string, traits map[string]interface{}) (*KratosUser, error) {
	updateRequest := client.UpdateIdentityBody{
		Traits: traits,
	}

	identity, resp, err := k.admin.IdentityAPI.UpdateIdentity(ctx, userID).
		UpdateIdentityBody(updateRequest).
		Execute()

	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to update user: status %d", resp.StatusCode)
	}

	return k.GetUser(ctx, identity.GetId())
}

func (k *KratosClient) DeleteUser(ctx context.Context, userID string) error {
	resp, err := k.admin.IdentityAPI.DeleteIdentity(ctx, userID).Execute()

	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to delete user: status %d", resp.StatusCode)
	}

	return nil
}

// Sessions Management
func (k *KratosClient) ListUserSessions(ctx context.Context, userID string) ([]client.Session, error) {
	sessions, resp, err := k.admin.IdentityAPI.ListIdentitySessions(ctx, userID).Execute()

	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list sessions: status %d", resp.StatusCode)
	}

	return sessions, nil
}

func (k *KratosClient) RevokeSession(ctx context.Context, sessionID string) error {
	resp, err := k.admin.IdentityAPI.DisableSession(ctx, sessionID).Execute()

	if err != nil {
		return fmt.Errorf("failed to revoke session: %w", err)
	}

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to revoke session: status %d", resp.StatusCode)
	}

	return nil
}

// Helper functions
func (k *KratosClient) isEmailVerified(ctx context.Context, userID string) bool {
	// В Kratos проверка верификации email происходит через recovery addresses
	// или через verification addresses (зависит от версии)

	// Простой способ - проверяем через identity metadata или verifiable addresses
	identity, resp, err := k.admin.IdentityAPI.GetIdentity(ctx, userID).Execute()
	if err != nil || resp.StatusCode != http.StatusOK {
		return false
	}

	// Проверяем verifiable addresses если они есть
	if addresses := identity.GetVerifiableAddresses(); len(addresses) > 0 {
		for _, addr := range addresses {
			if addr.GetVia() == "email" && addr.GetVerified() {
				return true
			}
		}
	}

	// Альтернативно проверяем recovery addresses
	if addresses := identity.GetRecoveryAddresses(); len(addresses) > 0 {
		for _, addr := range addresses {
			if addr.GetVia() == "email" {
				// Если recovery address существует, считаем email верифицированным
				return true
			}
		}
	}

	return false
}

func getStringFromTraits(traits interface{}, key string) string {
	if traits == nil {
		return ""
	}

	// Пробуем привести к map[string]interface{}
	if traitsMap, ok := traits.(map[string]interface{}); ok {
		if val, exists := traitsMap[key]; exists {
			if str, ok := val.(string); ok {
				return str
			}
		}
	}

	return ""
}

// Middleware для проверки сессии
func (k *KratosClient) SessionMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Получаем токен сессии из заголовка или cookie
			sessionToken := k.extractSessionToken(r)

			if sessionToken == "" {
				http.Error(w, "Session token required", http.StatusUnauthorized)
				return
			}

			// Валидируем сессию
			user, err := k.ValidateSession(r.Context(), sessionToken)
			if err != nil {
				http.Error(w, "Invalid session", http.StatusUnauthorized)
				return
			}

			// Добавляем пользователя в контекст
			ctx := context.WithValue(r.Context(), "user", user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func (k *KratosClient) extractSessionToken(r *http.Request) string {
	// Проверяем заголовок Authorization
	if auth := r.Header.Get("Authorization"); auth != "" {
		if strings.HasPrefix(auth, "Bearer ") {
			return strings.TrimPrefix(auth, "Bearer ")
		}
	}

	// Проверяем заголовок X-Session-Token
	if token := r.Header.Get("X-Session-Token"); token != "" {
		return token
	}

	// Проверяем cookie
	if cookie, err := r.Cookie("ory_kratos_session"); err == nil {
		return cookie.Value
	}

	return ""
}

// Utility функции для извлечения пользователя из контекста
func GetUserFromContext(ctx context.Context) (*KratosUser, bool) {
	user, ok := ctx.Value("user").(*KratosUser)
	return user, ok
}

func RequireUser(ctx context.Context) (*KratosUser, error) {
	user, ok := GetUserFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("user not found in context")
	}
	return user, nil
}
