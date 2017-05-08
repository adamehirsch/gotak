package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	jwtmiddleware "github.com/auth0/go-jwt-middleware"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/dgrijalva/jwt-go/request"
	"golang.org/x/crypto/bcrypt"
)

// checkJWTsignature will check a given token and verify that it was signed with the key and method specified below before passing access to its referenced Handler
var checkJWTsignature = jwtmiddleware.New(jwtmiddleware.Options{
	ValidationKeyGetter: jwtKeyFn,
	SigningMethod:       jwt.SigningMethodHS256,
	Debug:               false,
})

func jwtKeyFn(token *jwt.Token) (interface{}, error) {
	return []byte(opts.JWTkey), nil
}

func generateJWT(p *TakPlayer, m string) []byte {
	// At this point, presume the person's authenticated. Give them a token.
	token := jwt.New(jwt.SigningMethodHS256)

	// Create a map to store our claims
	claims := token.Claims.(jwt.MapClaims)

	claims["user"] = p.Username
	claims["exp"] = time.Now().Add(time.Hour * 24 * time.Duration(loginDays)).Unix()

	// sign the token
	tokenString, _ := token.SignedString([]byte(opts.JWTkey))
	thisJWT := TakJWT{
		JWT:     tokenString,
		Message: m,
	}
	JWTjson, _ := json.Marshal(thisJWT)
	return []byte(JWTjson)
}

// HashPassword uses bcrypt to produce a password hash suitable for storage
func HashPassword(pw string) []byte {
	password := []byte(pw)
	// Hashing the password with the default cost should be ample
	hashedPassword, err := bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}
	return hashedPassword
}

// VerifyPassword will verify ... wait for it ... that a password matches a hash
func VerifyPassword(pw string, hpw string) bool {
	if err := bcrypt.CompareHashAndPassword([]byte(hpw), []byte(pw)); err != nil {
		return false
	}
	return true

}

// authUser parses the username out of the JWT token and returns it to whoever's asking
func (env *DBenv) authUser(r *http.Request) (player *TakPlayer, err error) {
	token, _ := request.ParseFromRequest(r, request.AuthorizationHeaderExtractor, jwtKeyFn)
	claims := token.Claims.(jwt.MapClaims)
	username, ok := claims["user"].(string)
	if !ok {
		return nil, fmt.Errorf("no such player found: %v", username)
	}
	if player, err = env.db.RetrievePlayer(username); err != nil {
		return nil, err
	}
	return player, nil
}

// CanShow determines whether a given game can be shown to a given player
func (tg *TakGame) CanShow(p *TakPlayer) bool {
	switch {
	case tg.IsPublic:
		return true
	case tg.BlackPlayer == p.Username || tg.WhitePlayer == p.Username || tg.GameOwner == p.Username:
		return true
	default:
		return false
	}
}

// PlayersTurn determines whether a given player can make the next move
func (tg *TakGame) PlayersTurn(p *TakPlayer) bool {
	switch {
	case tg.BlackPlayer == p.Username && tg.IsBlackTurn == true:
		return true
	case tg.WhitePlayer == p.Username && tg.IsBlackTurn == false:
		return true
	default:
		return false
	}
}
