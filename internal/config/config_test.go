package config

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func restoreEnv(k string, v string) {
	if v == "" {
		_ = os.Unsetenv(k)
	} else {
		_ = os.Setenv(k, v)
	}
}

func TestLoad_WithTestingEnv(t *testing.T) {
	origEnv := os.Getenv("APP_ENV")
	defer restoreEnv("APP_ENV", origEnv)
	_ = os.Setenv("APP_ENV", "testing")

	cfg := Load()
	require.NotNil(t, cfg)
	assert.True(t, cfg.IsTesting())
	assert.False(t, cfg.IsProduction())
	assert.False(t, cfg.IsDevelopment())
	assert.NotNil(t, cfg.JWT.PrivateKey)
	assert.NotNil(t, cfg.JWT.PublicKey)
	assert.Equal(t, "8080", cfg.Server.Port)
	assert.Equal(t, "localhost", cfg.Server.Host)
	assert.Equal(t, "localhost", cfg.Database.Host)
	assert.Equal(t, "5432", cfg.Database.Port)
	assert.Equal(t, 25, cfg.Database.MaxConnections)
	assert.Equal(t, 5, cfg.Database.MaxIdleConns)
	assert.Equal(t, 12, cfg.Security.BCryptCost)
	assert.Equal(t, 5, cfg.Security.RateLimitPerSecond)
	assert.Equal(t, 24*time.Hour, cfg.JWT.AccessTokenDuration)
	assert.Equal(t, "banking-api", cfg.JWT.Issuer)
	assert.Equal(t, "http://regulator:9000/webhook", cfg.Regulator.WebhookURL)
	assert.Equal(t, 2, cfg.Regulator.RetryInitialSeconds)
	assert.Equal(t, 60, cfg.Regulator.RetryMaxSeconds)
}

func TestLoad_CORSAllowOrigins_EmptyDefaultsToStar(t *testing.T) {
	origAppEnv := os.Getenv("APP_ENV")
	origCORS := os.Getenv("CORS_ALLOW_ORIGINS")
	defer restoreEnv("APP_ENV", origAppEnv)
	defer restoreEnv("CORS_ALLOW_ORIGINS", origCORS)
	_ = os.Setenv("APP_ENV", "testing")
	_ = os.Unsetenv("CORS_ALLOW_ORIGINS")

	cfg := Load()
	require.Len(t, cfg.Server.CORSAllowOrigins, 1)
	assert.Equal(t, "*", cfg.Server.CORSAllowOrigins[0])
}

func TestLoad_CORSAllowOrigins_CommaSplitAndTrim(t *testing.T) {
	origAppEnv := os.Getenv("APP_ENV")
	origCORS := os.Getenv("CORS_ALLOW_ORIGINS")
	defer restoreEnv("APP_ENV", origAppEnv)
	defer restoreEnv("CORS_ALLOW_ORIGINS", origCORS)
	_ = os.Setenv("APP_ENV", "testing")
	_ = os.Setenv("CORS_ALLOW_ORIGINS", " https://a.com , https://b.com ")

	cfg := Load()
	require.Len(t, cfg.Server.CORSAllowOrigins, 2)
	assert.Equal(t, "https://a.com", cfg.Server.CORSAllowOrigins[0])
	assert.Equal(t, "https://b.com", cfg.Server.CORSAllowOrigins[1])
}

func TestDatabaseConfig_DSN(t *testing.T) {
	dc := DatabaseConfig{
		Host:    "dbhost",
		Port:    "5433",
		User:    "u",
		Password: "p",
		Name:    "mydb",
		SSLMode:  "require",
	}
	dsn := dc.DSN()
	assert.Contains(t, dsn, "host=dbhost")
	assert.Contains(t, dsn, "port=5433")
	assert.Contains(t, dsn, "user=u")
	assert.Contains(t, dsn, "password=p")
	assert.Contains(t, dsn, "dbname=mydb")
	assert.Contains(t, dsn, "sslmode=require")
}

func TestConfig_IsDevelopment_IsProduction_IsTesting(t *testing.T) {
	origAppEnv := os.Getenv("APP_ENV")
	defer restoreEnv("APP_ENV", origAppEnv)

	_ = os.Setenv("APP_ENV", "development")
	cfg := Load()
	assert.True(t, cfg.IsDevelopment())
	assert.False(t, cfg.IsProduction())
	assert.False(t, cfg.IsTesting())

	_ = os.Setenv("APP_ENV", "testing")
	cfg2 := Load()
	assert.True(t, cfg2.IsTesting())
	// IsProduction/IsDevelopment tested via struct to avoid Load() with production (requires JWT keys)
	c := &Config{}
	c.Server.Environment = "production"
	assert.True(t, c.IsProduction())
	assert.False(t, c.IsDevelopment())
	c.Server.Environment = "development"
	assert.True(t, c.IsDevelopment())
	assert.False(t, c.IsProduction())
}

func TestLoad_EnvOverrides(t *testing.T) {
	origPort := os.Getenv("SERVER_PORT")
	origDBHost := os.Getenv("DB_HOST")
	origAppEnv := os.Getenv("APP_ENV")
	defer restoreEnv("SERVER_PORT", origPort)
	defer restoreEnv("DB_HOST", origDBHost)
	defer restoreEnv("APP_ENV", origAppEnv)

	_ = os.Setenv("APP_ENV", "testing")
	_ = os.Setenv("SERVER_PORT", "9090")
	_ = os.Setenv("DB_HOST", "db.example.com")

	cfg := Load()
	assert.Equal(t, "9090", cfg.Server.Port)
	assert.Equal(t, "db.example.com", cfg.Database.Host)
}

// Test getIntEnv/getBoolEnv/getDurationEnv invalid values return defaults (cover err != nil branches)
func TestLoad_InvalidIntEnvUsesDefault(t *testing.T) {
	origAppEnv := os.Getenv("APP_ENV")
	origMaxConn := os.Getenv("DB_MAX_CONNECTIONS")
	defer restoreEnv("APP_ENV", origAppEnv)
	defer restoreEnv("DB_MAX_CONNECTIONS", origMaxConn)
	_ = os.Setenv("APP_ENV", "testing")
	_ = os.Setenv("DB_MAX_CONNECTIONS", "notanint")

	cfg := Load()
	assert.Equal(t, 25, cfg.Database.MaxConnections)
}

func TestLoad_InvalidBoolEnvUsesDefault(t *testing.T) {
	origAppEnv := os.Getenv("APP_ENV")
	origReq := os.Getenv("PASSWORD_REQUIRE_UPPERCASE")
	defer restoreEnv("APP_ENV", origAppEnv)
	defer restoreEnv("PASSWORD_REQUIRE_UPPERCASE", origReq)
	_ = os.Setenv("APP_ENV", "testing")
	_ = os.Setenv("PASSWORD_REQUIRE_UPPERCASE", "notabool")

	cfg := Load()
	assert.True(t, cfg.Security.RequireUppercase)
}

func TestLoad_InvalidDurationEnvUsesDefault(t *testing.T) {
	origAppEnv := os.Getenv("APP_ENV")
	origRead := os.Getenv("SERVER_READ_TIMEOUT")
	defer restoreEnv("APP_ENV", origAppEnv)
	defer restoreEnv("SERVER_READ_TIMEOUT", origRead)
	_ = os.Setenv("APP_ENV", "testing")
	_ = os.Setenv("SERVER_READ_TIMEOUT", "notaduration")

	cfg := Load()
	assert.Equal(t, 15*time.Second, cfg.Server.ReadTimeout)
}

// Test Load with JWT keys from env (covers loadKeysFromEnvVars, loadRSAPrivateKey, loadRSAPublicKey success paths)
func TestLoad_WithJWTKeysFromEnv(t *testing.T) {
	priv, pub, err := GenerateRSAKeyPair()
	require.NoError(t, err)
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	pubPKIX, _ := x509.MarshalPKIXPublicKey(pub)
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubPKIX})

	origAppEnv := os.Getenv("APP_ENV")
	origPriv := os.Getenv("JWT_PRIVATE_KEY")
	origPub := os.Getenv("JWT_PUBLIC_KEY")
	defer restoreEnv("APP_ENV", origAppEnv)
	defer restoreEnv("JWT_PRIVATE_KEY", origPriv)
	defer restoreEnv("JWT_PUBLIC_KEY", origPub)
	_ = os.Setenv("APP_ENV", "testing")
	_ = os.Setenv("JWT_PRIVATE_KEY", base64.StdEncoding.EncodeToString(privPEM))
	_ = os.Setenv("JWT_PUBLIC_KEY", base64.StdEncoding.EncodeToString(pubPEM))

	cfg := Load()
	require.NotNil(t, cfg.JWT.PrivateKey)
	require.NotNil(t, cfg.JWT.PublicKey)
}

// Test loadKeysFromEnvVars error paths (invalid base64)
func Test_loadKeysFromEnvVars_InvalidBase64(t *testing.T) {
	cfg := &Config{}
	_, _, err := cfg.loadKeysFromEnvVars("not-valid-base64!!!", "also-bad")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "JWT_PRIVATE_KEY")
}

func Test_loadKeysFromEnvVars_InvalidPublicKeyBase64(t *testing.T) {
	priv, _, _ := GenerateRSAKeyPair()
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	cfg := &Config{}
	_, _, err := cfg.loadKeysFromEnvVars(base64.StdEncoding.EncodeToString(privPEM), "not-valid-base64!!!")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "JWT_PUBLIC_KEY")
}

// Test loadRSAPrivateKey (same package can call unexported)
func Test_loadRSAPrivateKey_InvalidPEM(t *testing.T) {
	_, err := loadRSAPrivateKey([]byte("not pem data"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "PEM")
}

func Test_loadRSAPrivateKey_PKCS8(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	bytes, err := x509.MarshalPKCS8PrivateKey(priv)
	require.NoError(t, err)
	block := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: bytes})
	loaded, err := loadRSAPrivateKey(block)
	require.NoError(t, err)
	require.NotNil(t, loaded)
}

// PKCS8 with non-RSA key (ECDSA) returns "not an RSA private key"
func Test_loadRSAPrivateKey_PKCS8NotRSA(t *testing.T) {
	ecPriv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	bytes, err := x509.MarshalPKCS8PrivateKey(ecPriv)
	require.NoError(t, err)
	block := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: bytes})
	_, err = loadRSAPrivateKey(block)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not an RSA")
}

func Test_loadRSAPublicKey_InvalidPEM(t *testing.T) {
	_, err := loadRSAPublicKey([]byte("not pem data"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "PEM")
}

func Test_loadRSAPublicKey_Valid(t *testing.T) {
	priv, _, err := GenerateRSAKeyPair()
	require.NoError(t, err)
	pubPKIX, _ := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	block := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubPKIX})
	loaded, err := loadRSAPublicKey(block)
	require.NoError(t, err)
	require.NotNil(t, loaded)
}

func Test_GenerateRSAKeyPair(t *testing.T) {
	priv, pub, err := GenerateRSAKeyPair()
	require.NoError(t, err)
	require.NotNil(t, priv)
	require.NotNil(t, pub)
}

// loadRSAPublicKey returns "not an RSA public key" when PEM contains non-RSA key (e.g. ECDSA)
func Test_loadRSAPublicKey_NotRSA(t *testing.T) {
	ecPriv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	pubBytes, err := x509.MarshalPKIXPublicKey(&ecPriv.PublicKey)
	require.NoError(t, err)
	block := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes})
	_, err = loadRSAPublicKey(block)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not an RSA")
}
