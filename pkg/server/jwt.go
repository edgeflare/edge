package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/lestrrat-go/jwx/jwk"
)

func getKey(token *jwt.Token) (interface{}, error) {
	now := time.Now()

	if jwkCachedKeySet == nil || now.Sub(jwkLastFetched) > jwkCacheTTL {
		var err error
		jwkCachedKeySet, err = jwk.Fetch(context.Background(), conf.HTTP.Auth.JWT.JWKEndpoint)
		if err != nil {
			return nil, err
		}
		jwkLastFetched = now
	}

	keyID, ok := token.Header["kid"].(string)
	if !ok {
		return nil, errors.New("expecting JWT header to have a key ID in the kid field")
	}

	key, found := jwkCachedKeySet.LookupKeyID(keyID)
	if !found {
		return nil, fmt.Errorf("unable to find key %q", keyID)
	}

	var pubkey interface{}
	if err := key.Raw(&pubkey); err != nil {
		return nil, fmt.Errorf("unable to get the public key. Error: %s", err.Error())
	}

	return pubkey, nil
}

func verifyClientID(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		user, ok := c.Get("user").(*jwt.Token)
		if !ok {
			// Handle the case where the type assertion fails
			return c.JSON(http.StatusUnauthorized, "invalid user token")
		}

		claims, ok := user.Claims.(jwt.MapClaims)
		if !ok {
			// Handle the case where the claims type assertion fails
			return c.JSON(http.StatusUnauthorized, "invalid token claims")
		}

		azp, ok := claims["azp"].(string)
		if !ok {
			return c.JSON(http.StatusUnauthorized, "azp is missing from JWT")
		}

		if azp != conf.HTTP.Auth.JWT.ClientID {
			return c.JSON(http.StatusUnauthorized, "clientId/azp mismatch")
		}

		return next(c)
	}
}
