package theperfectgiftcard

import (
	"bytes"
	"crypto/rsa"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
)

const (
	cardNo = "5021234567890"
	pin    = "0000"
)

var rsaPrivateKey = &rsa.PrivateKey{
	PublicKey: rsa.PublicKey{
		N: fromBase10("9353930466774385905609975137998169297361893554149986716853295022578535724979677252958524466350471210367835187480748268864277464700638583474144061408845077"),
		E: 65537,
	},
	D: fromBase10("7266398431328116344057699379749222532279343923819063639497049039389899328538543087657733766554155839834519529439851673014800261285757759040931985506583861"),
	Primes: []*big.Int{
		fromBase10("98920366548084643601728869055592650835572950932266967461790948584315647051443"),
		fromBase10("94560208308847015747498523884063394671606671904944666360068158221458669711639"),
	},
}

func fromBase10(base10 string) *big.Int {
	i, ok := new(big.Int).SetString(base10, 10)
	if !ok {
		panic("bad number: " + base10)
	}
	return i
}

func testValue(t *testing.T, want interface{}, got interface{}) {
	if !reflect.DeepEqual(want, got) {
		t.Errorf("want: %v, got: %v", want, got)
	}
}

func testPin(t *testing.T, encryptedPin string) {
	decodedPin, err := hex.DecodeString(encryptedPin)
	if err != nil {
		t.Error(err)
	}

	decryptedPin, err := rsa.DecryptPKCS1v15(nil, rsaPrivateKey, decodedPin)
	if err != nil {
		t.Errorf("Decrypt hdnrequest failed. %v", err)
	}
	expectedhdnrequest := fmt.Sprintf("%s|%s", pin, randomNo)
	if !bytes.Equal(decryptedPin, []byte(expectedhdnrequest)) {
		t.Errorf("Decrypted hdnrequest: %s, want: %v", decryptedPin, expectedhdnrequest)
	}
}

func loadFixture(filename string) []byte {
	html, err := ioutil.ReadFile("fixtures/" + filename + ".html")
	if err != nil {
		panic(err)
	}
	return html
}

func TestGetCard(t *testing.T) {
	// prepare
	body["txtCardNumber"] = cardNo
	html := loadFixture("success")

	// setup
	c, err := NewClient()
	if err != nil {
		t.Error(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testValue(t, "POST", r.Method)
		testValue(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))
		r.ParseForm()
		for k, v := range body {
			t.Run(k, func(t *testing.T) {
				testValue(t, v, r.Form.Get(k))
			})
		}
		t.Run("hdnrequest", func(t *testing.T) {
			testPin(t, r.Form.Get("hdnrequest"))
		})
		w.Write(html)
	}))

	baseURL, _ := url.Parse(server.URL)
	c.BaseURL = baseURL
	c.publicKey = rsaPrivateKey.PublicKey

	card, res, err := c.GetCard(cardNo, pin)
	if err != nil {
		t.Error(err)
	}

	expectedCard := &Card{
		CardNo:           "50211234567890",
		AccountNo:        "000000000",
		LoadsToDate:      "$100.00",
		PurchasesToDate:  "-$54.32",
		AvailableBalance: "$12.34",
		PurchasedDate:    "1 Jan 2018",
		ExpiryDate:       "1 Jan 2021",
		Transactions: []Transaction{
			{
				Date:        "1 Jan 2018 12:04:45 PM",
				Details:     "Store Address",
				Description: "Refund - Store Address",
				Amount:      "$100.00",
				Balance:     "$100.00",
			},
			{
				Date:        "2 Jan 2018 07:50:53 PM",
				Details:     "Store A",
				Description: "Purchase - Store A",
				Amount:      "$12.34-",
				Balance:     "$56.78",
			},
		},
	}

	if res.StatusCode != http.StatusOK {
		t.Errorf("Response HTTP Status Code: %v, want: %v", res.StatusCode, http.StatusOK)
	}

	if !reflect.DeepEqual(html, res.Body) {
		t.Errorf("Response HTTP Body: %s, want: %s", html, res.Body)
	}

	if !reflect.DeepEqual(card, expectedCard) {
		t.Errorf("Card: %+v, want: %+v", card, expectedCard)
	}
}

func TestGetCardWithIncorrectPin(t *testing.T) {
	// prepare
	html := loadFixture("incorrect_cardno_or_pin")

	// setup
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(html)
	}))

	c, err := NewClient()
	if err != nil {
		t.Error(err)
	}
	baseURL, _ := url.Parse(server.URL)
	c.BaseURL = baseURL

	card, res, err := c.GetCard(cardNo, pin)
	if err == nil {
		t.Errorf("Expected an error to be thrown, got: nil")
	}

	expectedErrorMsg := "Invalid card number or password."
	if err.Error() != expectedErrorMsg {
		t.Errorf("Error Message: %s, want: %s", err.Error(), expectedErrorMsg)
	}

	if !reflect.DeepEqual(card, &Card{}) {
		t.Errorf("Card: %+v, want: %+v", card, &Card{})
	}

	if res.StatusCode != http.StatusUnauthorized {
		t.Errorf("Response HTTP Status Code: %v, want: %v", res.StatusCode, http.StatusUnauthorized)
	}

	if !reflect.DeepEqual(res.Body, html) {
		t.Errorf("Response HTTP Body: %s, want: %s", html, res.Body)
	}
}

func TestGetCardInternalError(t *testing.T) {
	// prepare
	html := loadFixture("application_error")

	// setup
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(html)
	}))

	c, _ := NewClient()
	baseURL, _ := url.Parse(server.URL)
	c.BaseURL = baseURL

	card, res, err := c.GetCard(cardNo, pin)
	if err == nil {
		t.Errorf("Expected an error to be thrown, got: nil")
	}

	expectedErrorMsg := "internal server error"
	if err.Error() != expectedErrorMsg {
		t.Errorf("Error Message: %s, want: %s", err.Error(), expectedErrorMsg)
	}

	if !reflect.DeepEqual(card, &Card{}) {
		t.Errorf("Card: %+v, want: %+v", card, &Card{})
	}

	if res.StatusCode != http.StatusInternalServerError {
		t.Errorf("Response HTTP Status Code: %v, want: %v", res.StatusCode, http.StatusInternalServerError)
	}

	if !reflect.DeepEqual(res.Body, html) {
		t.Errorf("Response HTTP Body: %s, want: %s", html, res.Body)
	}
}

func TestMakePublicKeyWithInvalidModulus(t *testing.T) {
	_, err := makePublicKey("invalid", exp)
	expectError := "invalid modulus"
	testValue(t, expectError, err.Error())
}

func TestMakePublicKeyWithInvalidExp(t *testing.T) {
	_, err := makePublicKey(modulus, "invalid")

	expectError := "strconv.ParseInt: parsing \"invalid\": invalid syntax"
	testValue(t, expectError, err.Error())
}

func TestInvalidPublicKey(t *testing.T) {
	c, _ := NewClient()
	c.publicKey = rsa.PublicKey{}
	card, resp, err := c.GetCard(cardNo, pin)
	testValue(t, &Card{}, card)
	testValue(t, &Response{}, resp)
	testValue(t, "crypto/rsa: missing public modulus", err.Error())
}
